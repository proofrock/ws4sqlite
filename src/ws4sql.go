/*
  Copyright (c) 2022-, Germano Rizzo <oss /AT/ germanorizzo /DOT/ it>

  Permission to use, copy, modify, and/or distribute this software for any
  purpose with or without fee is hereby granted, provided that the above
  copyright notice and this permission notice appear in all copies.

  THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
  WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
  MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
  ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
  WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
  ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
  OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/engines"
	"github.com/proofrock/ws4sql/structs"
	"github.com/wI2L/jettison"

	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/marcboeker/go-duckdb"
	_ "github.com/mattn/go-sqlite3"
)

const version = "ws4sql-v0.17dev3"

// Simply prints a header, parses the cli parameters and calls
// launch(), that is the real entry point. It's separate from the
// main method because launch() is called by the unit tests.
func main() {
	mllog.StdOutf("ws4sql %s", version)
	if sqliteVersion, err := engines.FLAV_SQLITE.GetVersion(); err != nil {
		mllog.Fatalf("getting sqlite version: %s", err.Error())
	} else {
		mllog.StdOutf("+ sqlite v%s", sqliteVersion)
	}
	if duckDBVersion, err := engines.FLAV_DUCKDB.GetVersion(); err != nil {
		mllog.Fatalf("getting duckdb version: %s", err.Error())
	} else {
		mllog.StdOutf("+ duckdb %s", duckDBVersion)
	}

	cfg := parseCLI()

	launch(cfg, false)
}

// A map with the database IDs as key, and the db struct as values.
var dbs map[string]structs.Db

// Fiber app, that serves the web service.
var app *fiber.App

// Actual entry point, called by main() and by the unit tests.
// Can be called multiple times, but the Fiber app must be
// terminated (see the Shutdown method in the tests).
func launch(cfg structs.Config, disableKeepAlive4Tests bool) {
	if len(cfg.Databases) == 0 && cfg.ServeDir == nil {
		mllog.Fatal("no database nor dir to serve specified")
	}

	// Let's create the web server
	app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          errHandler,
		// I use Jettyson to encode JSON because I want to be able to encode an empty resultset
		// but exclude a nil one from the resulting JSON; problem is, omitempty will exclude
		// both, so I use Jettison that allows an "omitnil" parameter that has the desired effect.
		JSONEncoder: jettison.Marshal,
		// This is because with keep alive on, in tests, the shutdown hangs until...
		// I think... some timeouts expire, but for a long time anyway. In normal
		// operations it's of course desirable.
		DisableKeepalive: disableKeepAlive4Tests,
		Network:          fiber.NetworkTCP,
	})
	// This intercepts the panics, and delegates them to the ErrorHandler.
	// See the comments to errHandler() to see why.
	app.Use(recover.New())

	// Later on, for each file created there will be a defer to remove it, unless this
	// guard is turned off
	var filesToDelete []string
	origWhenFatal := mllog.WhenFatal
	mllog.WhenFatal = func(msg string) {
		for _, ftd := range filesToDelete {
			os.Remove(ftd)
		}
		origWhenFatal(msg)
	}

	dbs = make(map[string]structs.Db)
	for i := range cfg.Databases {
		// beware: this variable is NOT modified in-place. Add a cfg.Databases[i] = database at the end, if need be
		database := ckConfig(cfg.Databases[i])
		dbId := *database.DatabaseDef.Id

		if _, ok := dbs[dbId]; ok {
			mllog.Fatalf("id '%s' already specified.", dbId)
		}

		mllog.StdOutf("  + Serving database '%s'", dbId)

		if !*database.DatabaseDef.InMemory && database.ToCreate {
			mllog.StdOut("  + File not present, it will be created")
		}

		if database.DatabaseDef.DisableWALMode != nil && !*database.DatabaseDef.DisableWALMode {
			mllog.StdOut("  + Using WAL")
		}

		if database.DatabaseDef.ReadOnly {
			mllog.StdOut("  + Read only")
		}

		if database.UseOnlyStoredStatements {
			mllog.StdOut("  + Strictly using only stored statements")
		}

		// Creates the mutex to be used to serialize the waiting time after a failed auth
		var mutex sync.Mutex
		database.Mutex = &mutex

		database.StoredStatsMap = make(map[string]string)

		for j := range database.StoredStatement {
			ss := database.StoredStatement[j]
			if ss.Id == "" || ss.Sql == "" {
				mllog.Fatalf("no ID or SQL specified for stored statement #%d in database '%s'", j, dbId)
			}
			database.StoredStatsMap[ss.Id] = ss.Sql
		}

		if len(database.StoredStatsMap) > 0 {
			mllog.StdOutf("  + With %d stored statements", len(database.StoredStatsMap))
		} else if database.UseOnlyStoredStatements {
			mllog.Fatalf("for db '%s', specified to use only stored statements but no one is provided", dbId)
		}

		// Opens the DB and adds it to the structure
		dbObj, err := database.ConnectionGetter()
		if err != nil {
			mllog.Fatal(err.Error())
		}
		// This method returns when the application exits. As per https://github.com/mattn/go-sqlite3/issues/1008,
		// it's not necessary to Close() the _db. The file remains consistent, and the pointers and locks are freed,
		// of course.

		// Executes a query on the DB, to create the file if not present
		// and report general errors as soon as possible.
		if _, err := dbObj.Exec("SELECT 1"); err != nil {
			mllog.Fatalf("accessing the database '%s': %s", dbId, err.Error())
		}

		// If this cycle will fail, I will have to clean up the created files
		if !*database.DatabaseDef.InMemory && database.ToCreate {
			filesToDelete = append(filesToDelete, *database.DatabaseDef.Path)
		}

		if database.ToCreate && len(database.InitStatements) > 0 {
			performInitStatements(database, dbObj, *database.DatabaseDef.InMemory)
		}

		database.Db = dbObj
		database.DbConn, err = dbObj.Conn(context.Background())
		if err != nil {
			mllog.Fatalf("in opening connection to %s: %s", dbId, err.Error())
		}

		// Parsing of the authentication
		if database.Auth != nil {
			parseAuth(&database)
		}

		// Parsing of the scheduled tasks
		// FIXME Fail if readonly?
		if database.Maintenance != nil && len(database.ScheduledTasks) > 0 {
			mllog.Fatalf("in %s: it's not possible to use both old maintenance and new scheduledTasks together. Move the maintenance task in the latter.", dbId)
		} else if database.Maintenance != nil {
			mllog.Warnf("in %s: \"maintenance\" node is deprecated, move it to \"scheduledTasks\"", dbId)
			database.ScheduledTasks = []structs.ScheduledTask{*database.Maintenance}
		}
		if len(database.ScheduledTasks) > 0 {
			parseTasks(&database)
		}

		if database.CORSOrigin != "" {
			mllog.StdOutf("  + CORS Origin set to %s", database.CORSOrigin)
		}

		dbs[dbId] = database
	}

	if cfg.ServeDir != nil {
		app.Static("", *cfg.ServeDir, fiber.Static{
			ByteRange: true,
		})
		mllog.StdOutf("- Serving directory '%s'", *cfg.ServeDir)
	}

	mllog.WhenFatal = origWhenFatal

	// Now all the maintenance plans for all the databases are parsed, so let's start the cron engine
	startTasks()

	// Register the handler
	for id := range dbs {
		db := dbs[id]

		var handlers []fiber.Handler

		if db.CORSOrigin != "" {
			handlers = append(handlers, cors.New(cors.Config{
				AllowMethods: "POST,OPTIONS",
				AllowOrigins: db.CORSOrigin,
			}))
		}

		if db.Auth != nil && strings.ToUpper(db.Auth.Mode) == authModeHttp {
			handlers = append(handlers, basicauth.New(basicauth.Config{
				Authorizer: func(user, password string) bool {
					if err := applyAuthCreds(&db, user, password); err != nil {
						// When unauthenticated waits for 1s, and doesn't parallelize, to hinder brute force attacks
						db.Mutex.Lock()
						time.Sleep(time.Second)
						db.Mutex.Unlock()
						mllog.Errorf("credentials not valid for user '%s'", user)
						return false
					}
					return true
				},
				Unauthorized: func(c *fiber.Ctx) error {
					if db.Auth.CustomErrorCode != nil {
						return c.Status(*db.Auth.CustomErrorCode).SendString("Unauthorized")
					}
					return c.SendStatus(fiber.StatusUnauthorized)
				},
			}))
		}

		handlers = append(handlers, handler(*db.DatabaseDef.Id))

		app.Post(fmt.Sprintf("/%s", *db.DatabaseDef.Id), handlers...)

		if db.CORSOrigin != "" {
			app.Options(fmt.Sprintf("/%s", *db.DatabaseDef.Id), handlers...)
		}
	}

	// Actually start the web server, finally
	conn := fmt.Sprint(cfg.Bindhost, ":", cfg.Port)
	mllog.StdOut("- Web Service listening on ", conn)
	if err := app.Listen(conn); err != nil {
		mllog.Fatal(err.Error())
	}
}

func performInitStatements(database structs.Db, dbObj *sql.DB, isMemory bool) {
	// This is implemented in its own method to allow the defer to run ASAP

	// Execute non-concurrently
	database.Mutex.Lock()
	defer database.Mutex.Unlock()

	for j := range database.InitStatements {
		if _, err := dbObj.Exec(database.InitStatements[j]); err != nil {
			if !isMemory {
				// I fail and abort, so remove the leftover file
				// TODO should I remove the wal files?
				dbObj.Close()
				os.Remove(*database.DatabaseDef.Path)
			}
			mllog.Fatalf("in init statement #%d for database '%s': %s", j+1, *database.DatabaseDef.Id, err.Error())
		}
	}
	mllog.StdOutf("  + %d init statements performed", len(database.InitStatements))
}
