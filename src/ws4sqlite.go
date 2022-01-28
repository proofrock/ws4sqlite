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
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/proofrock/crypgo"
	mllog "github.com/proofrock/go-mylittlelogger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/mitchellh/go-homedir"

	_ "github.com/mattn/go-sqlite3"

	"gopkg.in/yaml.v2"
)

const version = "0.9.0"

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

	return c.Status(ret.Code).JSON(ret)
}

func main() {
	mllog.StdOut("ws4sqlite ", version)

	cfgDir := flag.String("cfgDir", ".", "Directory where to look for the config.yaml file (default: current dir).")
	bindHost := flag.String("bindHost", "0.0.0.0", "The host to bind (default: 0.0.0.0).")
	port := flag.Int("port", 12321, "Port for the web service (default: 12321).")
	version := flag.Bool("version", false, "Display the version number")

	flag.Parse()

	if *version {
		os.Exit(0)
	}

	var err error

	*cfgDir, err = homedir.Expand(*cfgDir)
	if err != nil {
		mllog.Fatal("in expanding config file path: ", err.Error())
	}

	cfgData, err := os.ReadFile(filepath.Join(*cfgDir, "config.yaml"))
	if err != nil {
		mllog.Fatal("in reading config file: ", err.Error())
	}

	var cfg config
	if err = yaml.Unmarshal(cfgData, &cfg); err != nil {
		mllog.Fatal("in parsing config file: ", err.Error())
	}

	cfg.Bindhost = *bindHost
	cfg.Port = *port

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
				Authorizer: func(user, pass string) bool {
					if err := applyAuthCreds(&db, user, pass); err != nil {
						db.Mutex.Lock() // When unauthenticated waits for 2s, and doesn't parallelize, to hinder brute force attacks
						time.Sleep(2 * time.Second)
						db.Mutex.Unlock()
						mllog.Warnf("credentials not valid for user '%s'", user)
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

// Scans the values for a db request and encrypts them as needed
func encrypt(encoder requestItemCrypto, values map[string]interface{}) error {
	for i := range encoder.Columns {
		sval, ok := values[encoder.Columns[i]].(string)
		if !ok {
			return errors.New("attempting to encrypt a non-string column")
		}
		var eval string
		var err error
		if encoder.CompressionLevel < 1 {
			eval, err = crypgo.Encrypt(encoder.Pwd, sval)
		} else if encoder.CompressionLevel < 20 {
			eval, err = crypgo.CompressAndEncrypt(encoder.Pwd, sval, encoder.CompressionLevel)
		} else {
			return errors.New("compression level is in the range 0-19")
		}
		if err != nil {
			return err
		}
		values[encoder.Columns[i]] = eval
	}
	return nil
}

// Scans the results from a db request and decrypts them as needed
func decrypt(decoder requestItemCrypto, results map[string]interface{}) error {
	if decoder.CompressionLevel > 0 {
		return errors.New("cannot specify compression level for decryption")
	}
	for i := range decoder.Columns {
		sval, ok := results[decoder.Columns[i]].(string)
		if !ok {
			return errors.New("attempting to decrypt a non-string column")
		}
		dval, err := crypgo.Decrypt(decoder.Pwd, sval)
		if err != nil {
			return err
		}
		results[decoder.Columns[i]] = dval
	}
	return nil
}

// For a single query item, deals with a failure, determining if it must invalidate all of the transaction
// or just report an error in the single query
func reportError(err error, code int, idx int, noFail bool, results []responseItem) []responseItem {
	if !noFail {
		panic(newWSError(idx, code, err.Error()))
	}
	return append(results, ResItem4Error(capitalize(err.Error())))
}

func processWithResultSet(tx *sql.Tx, q string, decoder *requestItemCrypto, values map[string]interface{}) (responseItem, error) {
	resultSet := make([]map[string]interface{}, 0)

	rows, err := tx.Query(q, vals2nameds(values)...)
	if err != nil {
		return ResItemEmpty(), err
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return ResItemEmpty(), err
	}
	for rows.Next() {
		values := make([]interface{}, len(colNames))
		scans := make([]interface{}, len(colNames))
		for i := range values {
			scans[i] = &values[i]
		}
		if err = rows.Scan(scans...); err != nil {
			return ResItemEmpty(), err
		}

		toAdd := make(map[string]interface{})
		for i := range values {
			toAdd[colNames[i]] = values[i]
		}

		if decoder != nil {
			if err := decrypt(*decoder, toAdd); err != nil {
				return ResItemEmpty(), err
			}
		}
		resultSet = append(resultSet, toAdd)
	}

	if err = rows.Err(); err != nil {
		return ResItemEmpty(), err
	}

	return ResItem4Query(resultSet), nil
}

func processForExec(tx *sql.Tx, q string, values map[string]interface{}) (responseItem, error) {
	qres, err := tx.Exec(q, vals2nameds(values)...)
	if err != nil {
		return ResItemEmpty(), err
	}

	rAff, err := qres.RowsAffected()
	if err != nil {
		return ResItemEmpty(), err
	}

	return ResItem4Statement(rAff), nil
}

func processForExecBatch(tx *sql.Tx, q string, valuesBatch []map[string]interface{}) (responseItem, error) {
	ps, err := tx.Prepare(q)
	if err != nil {
		return ResItemEmpty(), err
	}
	defer ps.Close()

	var rAffs []int64
	for i := range valuesBatch {
		qres, err := ps.Exec(vals2nameds(valuesBatch[i])...)
		if err != nil {
			return ResItemEmpty(), err
		}

		rAff, err := qres.RowsAffected()
		if err != nil {
			return ResItemEmpty(), err
		}

		rAffs = append(rAffs, rAff)
	}

	return ResItem4Batch(rAffs), nil
}

func handler(c *fiber.Ctx) error {
	var body request
	if err := c.BodyParser(&body); err != nil {
		panic(newWSError(-1, fiber.StatusBadRequest, "in parsing body: %s", err.Error()))
	}

	databaseId := c.Params("databaseId")
	if databaseId == "" {
		panic(newWSError(-1, fiber.StatusNotFound, "missing database ID"))
	}

	_db, found := dbs[databaseId]
	if !found {
		panic(newWSError(-1, fiber.StatusNotFound, "database with ID '%s' not found", databaseId))
	}
	db := _db

	if db.Auth != nil && strings.ToUpper(db.Auth.Mode) == authModeInline {
		if err := applyAuth(&db, &body); err != nil {
			db.Mutex.Lock() // When unauthenticated waits for 2s, and doesn't parallelize, to hinder brute force attacks
			time.Sleep(2 * time.Second)
			db.Mutex.Unlock()
			panic(newWSError(-1, fiber.StatusUnauthorized, err.Error()))
		}
	}

	if body.Transaction == nil || len(body.Transaction) == 0 {
		panic(newWSError(-1, fiber.StatusBadRequest, "missing statements list ('transaction' node)"))
	}

	var ret response

	dbc, err := db.Db.Conn(context.Background())
	if err != nil {
		panic(newWSError(-1, fiber.StatusInternalServerError, err.Error()))
	}
	defer dbc.Close()

	tx, err := dbc.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: db.ReadOnly})
	if err != nil {
		panic(newWSError(-1, fiber.StatusInternalServerError, err.Error()))
	}

	tainted := true // if I reach the end of the method, I switch this to false to signal success
	defer func() {
		if tainted {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for i := range body.Transaction {
		if body.Transaction[i].Query == "" && body.Transaction[i].Statement == "" {
			ret.Results = reportError(errors.New("neither query nor statement specified"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		if body.Transaction[i].Query != "" && body.Transaction[i].Statement != "" {
			ret.Results = reportError(errors.New("cannot specify both query and statement"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		hasResultSet := body.Transaction[i].Query != ""

		if !hasResultSet {
			body.Transaction[i].Query = body.Transaction[i].Statement
		}

		if len(body.Transaction[i].Values) != 0 && len(body.Transaction[i].ValuesBatch) != 0 {
			ret.Results = reportError(errors.New("cannot specify both values and valuesBatch"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		if hasResultSet && len(body.Transaction[i].ValuesBatch) != 0 {
			ret.Results = reportError(errors.New("cannot specify valuesBatch for queries (only for statements)"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		hasBatch := len(body.Transaction[i].ValuesBatch) != 0

		var q string
		if strings.HasPrefix(body.Transaction[i].Query, "#") {
			var ok bool
			q, ok = db.StoredStatsMap[body.Transaction[i].Query[1:]]
			if !ok {
				ret.Results = reportError(errors.New("a stored statement is required, but did not find it"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}
		} else {
			if db.UseOnlyStoredStatements {
				ret.Results = reportError(errors.New("configured to serve only stored statements, but SQL is passed"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}
			q = body.Transaction[i].Query
		}

		if hasBatch {
			var valuesBatch []map[string]interface{}
			for i2 := range body.Transaction[i].ValuesBatch {
				values, err := raw2vals(body.Transaction[i].ValuesBatch[i2])
				if err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}

				if body.Transaction[i].Encoder != nil {
					if err := encrypt(*body.Transaction[i].Encoder, values); err != nil {
						ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
						continue
					}
				}

				valuesBatch = append(valuesBatch, values)
			}

			retE, err := processForExecBatch(tx, q, valuesBatch)
			if err != nil {
				ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}

			ret.Results = append(ret.Results, retE)
		} else {
			values, err := raw2vals(body.Transaction[i].Values)
			if err != nil {
				ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}

			if body.Transaction[i].Encoder != nil {
				if err := encrypt(*body.Transaction[i].Encoder, values); err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}
			}

			if hasResultSet {
				// Externalized in a func so that defer rows.Close() actually runs
				retWR, err := processWithResultSet(tx, q, body.Transaction[i].Decoder, values)
				if err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}

				ret.Results = append(ret.Results, retWR)
			} else {
				retE, err := processForExec(tx, q, values)
				if err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}

				ret.Results = append(ret.Results, retE)
			}
		}
	}

	tainted = false
	return c.JSON(ret)
}
