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

package engines

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

type duckdbEngine struct{}

func (s *duckdbEngine) GetVersion() (string, error) {
	dbObj, err := sql.Open("duckdb", "")
	defer func() { dbObj.Close() }()
	if err != nil {
		return "", err
	}
	row := dbObj.QueryRow("SELECT version()")
	var ver string
	err = row.Scan(&ver)
	if err != nil {
		return "", err
	}
	return ver, nil
}

func (s *duckdbEngine) GetDefaultIsolationLevel() sql.IsolationLevel {
	return sql.LevelDefault
}

func (s *duckdbEngine) CheckRequest(body structs.Request) *structs.WsError {
	for i, req := range body.Transaction {
		if req.NoFail {
			return utils.Ptr(structs.NewWSError(i, fiber.StatusBadRequest, "'noFail' not supported in DUCKDB type"))
		}
	}
	return nil
}

func (s *duckdbEngine) CheckConfig(dbConfig structs.Db) structs.Db {
	if dbConfig.DatabaseDef.DisableWALMode != nil {
		mllog.Fatal("cannot specify WAL mode for DuckDB")
	}

	if *dbConfig.DatabaseDef.InMemory {
		if dbConfig.DatabaseDef.Id == nil {
			mllog.Fatal("missing explicit Id for In-Memory db: ", dbConfig.ConfigFilePath)
		}
		dbConfig.DatabaseDef.Path = utils.Ptr("")
	} else {
		if *dbConfig.DatabaseDef.Path == "" {
			mllog.Fatal("no path specified for db: ", dbConfig.ConfigFilePath)
		}

		// resolves '~' // FIXME necessary?
		dbConfig.DatabaseDef.Path = utils.Ptr(utils.ExpandHomeDir(*dbConfig.DatabaseDef.Path, "database file"))
		if dbConfig.DatabaseDef.Id == nil {
			dbConfig.DatabaseDef.Id = utils.Ptr(
				strings.TrimSuffix(
					filepath.Base(*dbConfig.DatabaseDef.Path),
					filepath.Ext(*dbConfig.DatabaseDef.Path),
				),
			)
			if len(*dbConfig.DatabaseDef.Id) == 0 {
				mllog.Fatal("base filename cannot be empty in ", dbConfig.ConfigFilePath)
			}
		}
	}

	// Is the database new? Later I'll have to create the InitStatements
	dbConfig.ToCreate = *dbConfig.DatabaseDef.InMemory || !utils.FileExists(*dbConfig.DatabaseDef.Path)

	// Compose the connection string
	var connString strings.Builder
	connString.WriteString(*dbConfig.DatabaseDef.Path)
	var options []string
	if dbConfig.DatabaseDef.ReadOnly {
		options = append(options, "ACCESS_MODE=READ_ONLY")
	}
	if len(options) > 0 {
		connString.WriteRune('?')
		connString.WriteString(strings.Join(options, "&"))
	}

	dbConfig.ConnectionGetter = func() (*sql.DB, error) { return sql.Open("duckdb", connString.String()) }

	return dbConfig
}
