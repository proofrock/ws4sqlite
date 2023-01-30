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
	"database/sql"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/mitchellh/go-homedir"
	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/wI2L/jettison"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "modernc.org/sqlite"
)

const version = "0.13.0_dev1"

func getSQLiteVersion() (string, error) {
	dbObj, err := sql.Open("sqlite", ":memory:")
	defer dbObj.Close()
	if err != nil {
		return "", err
	}
	row := dbObj.QueryRow("SELECT sqlite_version()")
	var ver string
	err = row.Scan(&ver)
	if err != nil {
		return "", err
	}
	return ver, nil
}

// Simply prints a header, parses the cli parameters and calls
// launch(), that is the real entry point. It's separate from the
// main method because launch() is called by the unit tests.
func main() {
	mllog.StdOut("ws4sqlite ", version)
	sqliteVersion, err := getSQLiteVersion()
	if err != nil {
		mllog.Fatalf("getting SQLite version: %s", err.Error())
	}
	mllog.StdOut("- Based on SQLite v" + sqliteVersion)

	cfg := parseCLI()

	launch(cfg, false)
}

// A map with the database IDs as key, and the db struct as values.
var dbs map[string]db

// Fiber app, that serves the web service.
var app *fiber.App

// Actual entry point, called by main() and by the unit tests.
// Can be called multiple times, but the Fiber app must be
// terminated (see the Shutdown method in the tests).
func launch(cfg config, disableKeepAlive4Tests bool) {
	var err error

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

	dbs = make(map[string]db)
	for i := range cfg.Databases {
		// beware: this variable is NOT modified in-place. Add a cfg.Databases[i] = database at the end, if need be
		database := cfg.Databases[i]

		if database.Id == "" {
			mllog.Fatalf("no id specified for db #%d.", i)
		}

		if _, ok := dbs[database.Id]; ok {
			mllog.Fatalf("id '%s' already specified.", database.Id)
		}

		// FIXME check if this is enough to consider it in-memory
		isMemory := strings.Contains(database.Path, ":memory:")

		if database.Path == "" {
			mllog.Fatalf("no path specified for db '%s'.", database.Id)
		}

		if !isMemory {
			// Resolves '~'
			if database.Path, err = homedir.Expand(database.Path); err != nil {
				mllog.Fatal("in expanding db file path: ", err.Error())
			}
		}

		// Is the database new? Later I'll have to create the InitStatements
		toCreate := isMemory || !fileExists(database.Path)

		connString := database.Path
		var options []string
		if database.ReadOnly {
			// Several ways to be read-only...
			options = append(options, "_pragma=query_only(true)")
		}
		if !database.DisableWALMode {
			options = append(options, "_pragma=journal_mode(WAL)")
		}
		if len(options) > 0 {
			connString = connString + "?" + strings.Join(options, "&")
		}

		mllog.StdOutf("- Serving database '%s' from %s", database.Id, connString)

		if database.HasConfigFile {
			mllog.StdOut("  + Parsed companion config file")
		} else {
			mllog.StdOut("  + No config file loaded, using defaults")
		}

		if database.ReadOnly && toCreate && len(database.InitStatements) > 0 {
			mllog.Fatalf("'%s': a new db cannot be read only and have init statement", database.Id)
		}

		if !isMemory && toCreate {
			mllog.StdOut("  + File not present, it will be created")
		}

		if !database.DisableWALMode {
			mllog.StdOut("  + Using WAL")
		}

		if database.ReadOnly {
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
				mllog.Fatalf("no ID or SQL specified for stored statement #%d in database '%s'", j, database.Id)
			}
			database.StoredStatsMap[ss.Id] = ss.Sql
		}

		if len(database.StoredStatsMap) > 0 {
			mllog.StdOutf("  + With %d stored statements", len(database.StoredStatsMap))
		} else if database.UseOnlyStoredStatements {
			mllog.Fatalf("for db '%s', specified to use only stored statements but no one is provided", database.Id)
		}

		// Opens the DB and adds it to the structure
		dbObj, err := sql.Open("sqlite", connString)
		if err != nil {
			mllog.Fatal(err.Error())
		}
		// This method returns when the application exits. As per https://github.com/mattn/go-sqlite3/issues/1008,
		// it's not necessary to Close() the _db. The file remains consistent, and the pointers and locks are freed,
		// of course.

		// Executes a query on the DB, to create the file if not present
		// and report general errors as soon as possible.
		if _, err := dbObj.Exec("SELECT 1"); err != nil {
			mllog.Fatalf("accessing the database '%s': %s", database.Id, err.Error())
		}

		// If this cycle will fail, I will have to clean up the created files
		if toCreate && !isMemory {
			filesToDelete = append(filesToDelete, database.Path)
		}

		if toCreate && len(database.InitStatements) > 0 {
			performInitStatements(database, dbObj, isMemory)
		}

		database.Db = dbObj

		// Parsing of the authentication
		if database.Auth != nil {
			parseAuth(&database)
		}

		// Parsing of the maintenance plan
		if database.Maintenance != nil {
			parseMaint(&database)
		}

		if database.CORSOrigin != "" {
			mllog.StdOutf("  + CORS Origin set to %s", database.CORSOrigin)
		}

		dbs[database.Id] = database
	}

	if cfg.ServeDir != nil {
		app.Static("", *cfg.ServeDir, fiber.Static{
			ByteRange: true,
		})
		mllog.StdOutf("- Serving directory '%s'", *cfg.ServeDir)
	}

	mllog.WhenFatal = origWhenFatal

	// Now all the maintenance plans for all the databases are parsed, so let's start the cron engine
	startMaint(dbs)

	// Register the handler
	for id, _ := range dbs {
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

		handlers = append(handlers, handler(db.Id))

		app.Post(fmt.Sprintf("/%s", db.Id), handlers...)

		if db.CORSOrigin != "" {
			app.Options(fmt.Sprintf("/%s", db.Id), handlers...)
		}
	}

	// Actually start the web server, finally
	conn := fmt.Sprint(cfg.Bindhost, ":", cfg.Port)
	mllog.StdOut("- Web Service listening on ", conn)
	if err := app.Listen(conn); err != nil {
		mllog.Fatal(err.Error())
	}
}

func performInitStatements(database db, dbObj *sql.DB, isMemory bool) {
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
				os.Remove(database.Path)
			}
			mllog.Fatalf("in init statement #%d for database '%s': %s", j+1, database.Id, err.Error())
		}
	}
	mllog.StdOutf("  + %d init statements performed", len(database.InitStatements))
}
