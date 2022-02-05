/*
  Copyright (c) 2022-, Germano Rizzo <oss@germanorizzo.it>

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
	"os"
	"strings"
	"sync"
	"time"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/wI2L/jettison"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mitchellh/go-homedir"

	_ "github.com/mattn/go-sqlite3"
)

const version = "0.11pre1"

// Catches the panics and converts the argument in a struct that Fiber uses to
// signal the error, setting the response code and the JSON that is actually returned
// with all its properties.
// It uses <panic> and the recover middleware to manage errors because it's the only
// way I know to let a custom structure/error arrive here; the standard way can only
// wrap a string.
func errHandler(c *fiber.Ctx, err error) error {
	var ret wsError

	if fe, ok := err.(*fiber.Error); ok {
		ret = newWSError(-1, fe.Code, capitalize(fe.Error()))
	} else if wse, ok := err.(wsError); ok {
		ret = wse
	} else {
		ret = newWSError(-1, fiber.StatusInternalServerError, capitalize(err.Error()))
	}

	bytes, err := jettison.Marshal(ret)
	if err != nil {
		// FIXME endless recursion? Unlikely, if jettison does its job
		return errHandler(c, newWSError(-1, fiber.StatusInternalServerError, err.Error()))
	}

	c.Response().Header.Add("Content-Type", "application/json")

	return c.Status(ret.Code).Send(bytes)
}

func main() {
	mllog.StdOut("ws4sqlite ", version)

	cfg := parseCLI()

	launch(cfg, false)
}

var dbs map[string]db
var app *fiber.App

func launch(cfg config, disableKeepAlive4Tests bool) {
	var err error

	if len(cfg.Databases) == 0 {
		mllog.Fatal("no database specified")
	}

	dbs = make(map[string]db)
	for i := range cfg.Databases {
		if cfg.Databases[i].Id == "" {
			mllog.Fatalf("no id specified for db #%d.", i)
		}

		if _, ok := dbs[cfg.Databases[i].Id]; ok {
			mllog.Fatalf("id '%s' already specified.", cfg.Databases[i].Id)
		}

		isMemory := strings.Contains(cfg.Databases[i].Path, ":memory:")

		if cfg.Databases[i].Path == "" {
			mllog.Fatalf("no path specified for db '%s'.", cfg.Databases[i].Id)
		}
		if !isMemory {
			if cfg.Databases[i].Path, err = homedir.Expand(cfg.Databases[i].Path); err != nil {
				mllog.Fatal("in expanding db file path: ", err.Error())
			}
		}

		toCreate := isMemory || !fileExists(cutUntil(cfg.Databases[i].Path, "?"))

		url := cfg.Databases[i].Path
		var items []string
		if cfg.Databases[i].ReadOnly {
			items = append(items, "mode=ro", "immutable=1", "_query_only=1")
		}
		if !cfg.Databases[i].DisableWALMode {
			items = append(items, "_journal=WAL")
		}
		if len(items) > 0 {
			var initiator string // the url may already contain some parameters
			if strings.Contains(url, "?") {
				initiator = "&"
			} else {
				initiator = "?"
			}
			url = url + initiator + strings.Join(items, "&")
		}

		mllog.StdOutf("- Serving database '%s' from %s", cfg.Databases[i].Id, url)

		if toCreate {
			mllog.StdOut("  + File not present, it will be created")
		}

		if !cfg.Databases[i].DisableWALMode {
			mllog.StdOut("  + Using WAL")
		}

		if cfg.Databases[i].ReadOnly && toCreate && len(cfg.Databases[i].InitStatements) > 0 {
			mllog.Fatalf("'%s': a new db cannot be read only and have init statement", cfg.Databases[i].Id)
		}

		if cfg.Databases[i].ReadOnly {
			mllog.StdOut("  + Read only")
		}

		if cfg.Databases[i].UseOnlyStoredStatements {
			mllog.StdOut("  + Strictly using stored statements")
		}

		cfg.Databases[i].StoredStatsMap = make(map[string]string)

		for i2 := range cfg.Databases[i].StoredStatement {
			if cfg.Databases[i].StoredStatement[i2].Id == "" || cfg.Databases[i].StoredStatement[i2].Sql == "" {
				mllog.Fatalf("no ID or SQL specified for stored statement #%d in database '%s'", i2, cfg.Databases[i].Id)
			}
			cfg.Databases[i].StoredStatsMap[cfg.Databases[i].StoredStatement[i2].Id] = cfg.Databases[i].StoredStatement[i2].Sql
		}

		if len(cfg.Databases[i].StoredStatsMap) > 0 {
			mllog.StdOutf("  + With %d stored statements", len(cfg.Databases[i].StoredStatsMap))
		} else if cfg.Databases[i].UseOnlyStoredStatements {
			mllog.Fatalf("for db '%s', specified to use only stored statements but no one is provided", cfg.Databases[i].Id)
		}

		// Opens the DB and adds it to the structure
		_db, err := sql.Open("sqlite3", url)
		if err != nil {
			mllog.Fatal(err.Error())
		}
		// This method returns when the application exits. As per https://github.com/mattn/go-sqlite3/issues/1008,
		// it's not necessary to Close() the _db. The file remains consistent, and the pointers and locks are freed, of course.

		// For concurrent writes, see  https://stackoverflow.com/questions/35804884/sqlite-concurrent-writing-performance
		// If WAL is disabled, disable concurrency; see https://sqlite.org/wal.html
		if !cfg.Databases[i].ReadOnly || cfg.Databases[i].DisableWALMode {
			_db.SetMaxOpenConns(1)
		}

		// Executes a query on the DB, to create the file if not present
		// and report general errors as soon as possible.
		if _, err := _db.Exec("SELECT 1"); err != nil {
			mllog.Fatalf("accessing the database '%s': %s", cfg.Databases[i].Id, err.Error())
		}

		if toCreate && len(cfg.Databases[i].InitStatements) > 0 {
			for i2 := range cfg.Databases[i].InitStatements {
				if _, err := _db.Exec(cfg.Databases[i].InitStatements[i2]); err != nil {
					if !isMemory {
						os.Remove(cutUntil(url, "?"))
					}
					mllog.Fatalf("in init statement %d for database '%s': %s", i2+1, cfg.Databases[i].Id, err.Error())
				}
			}
			mllog.StdOutf("  + %d init statements performed", len(cfg.Databases[i].InitStatements))
		}

		// Creates the mutex to be used to serialize the waith after a failed auth
		var mutex sync.Mutex
		cfg.Databases[i].Mutex = &mutex

		cfg.Databases[i].Db = _db

		if cfg.Databases[i].Auth != nil {
			parseAuth(&cfg.Databases[i])
		}

		if cfg.Databases[i].CORSOrigin != "" {
			mllog.StdOut("  + CORS Origin set to ", cfg.Databases[i].CORSOrigin)
		}

		if cfg.Databases[i].Maintenance != nil {
			if cfg.Databases[i].ReadOnly {
				mllog.Fatalf("'%s': a db cannot be read only and have a maintenance plan", cfg.Databases[i].Id)
			}
			parseMaint(&cfg.Databases[i])
		}

		dbs[cfg.Databases[i].Id] = cfg.Databases[i]
	}

	startMaint()

	app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          errHandler,
		DisableKeepalive:      disableKeepAlive4Tests,
	})
	app.Use(recover.New())

	// See if CORS is needed for the specifies databases, and for each one
	// adds an instance of the CORS middleware
	for k := range dbs {
		db := dbs[k]
		// in the middlewares, c.Param("databaseId") doesn't work, because they are outside an handler
		// so we just use c.Path()[1:]
		if db.CORSOrigin != "" {
			app.Use(cors.New(cors.Config{
				Next: func(c *fiber.Ctx) bool {
					switch c.Method() {
					case "POST":
						return c.Path()[1:] != db.Id
					case "OPTIONS":
						return db.CORSOrigin != "*" && db.CORSOrigin != c.Get("Origin")
					default:
						return true
					}
				},
				AllowMethods: "POST",
				AllowOrigins: db.CORSOrigin,
			}))
		}
		if db.Auth != nil && strings.ToUpper(db.Auth.Mode) == authModeHttp {
			app.Use(basicauth.New(basicauth.Config{
				Next: func(c *fiber.Ctx) bool {
					return c.Path()[1:] != db.Id
				},
				Authorizer: func(user, password string) bool {
					if err := applyAuthCreds(&db, user, password); err != nil {
						db.Mutex.Lock() // When unauthenticated waits for 2s, and doesn't parallelize, to hinder brute force attacks
						time.Sleep(2 * time.Second)
						db.Mutex.Unlock()
						mllog.Errorf("credentials not valid for user '%s'", user)
						return false
					}
					return true
				},
			}))
		}
	}

	app.Post("/:databaseId", handler)

	conn := fmt.Sprint(cfg.Bindhost, ":", cfg.Port)
	mllog.StdOut("- Web Service listening on ", conn)
	if err := app.Listen(conn); err != nil {
		mllog.Fatal(err.Error())
	}
}
