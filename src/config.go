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
	"path/filepath"
	"strings"

	mllog "github.com/proofrock/go-mylittlelogger"
)

// In this file, a config passed by an user on the commandline is checked and
// if necessary normalized (default values, ecc).

func ckConfig(dbConfig db) db {
	mllog.StdOutf("- Parsing config file: %s", dbConfig.ConfigFilePath)

	if dbConfig.DatabaseDef.Type == nil {
		dbConfig.DatabaseDef.Type = Ptr("SQLITE")
		mllog.StdOutf("  + No type specified, assuming SQLITE")
	}

	var ret db
	switch *dbConfig.DatabaseDef.Type {
	case "SQLITE":
		ret = ckConfigSQLITE(dbConfig)
	default:
		mllog.Fatal("invalid type: ", *dbConfig.DatabaseDef.Type)
		return dbConfig // never reached
	}

	if ret.DatabaseDef.ReadOnly && dbConfig.ToCreate && len(dbConfig.InitStatements) > 0 {
		mllog.Fatal("a new db cannot be read only and have init statement: ", dbConfig.ConfigFilePath)
	}

	return ret
}

func ckConfigSQLITE(dbConfig db) db {
	if dbConfig.DatabaseDef.InMemory == nil {
		dbConfig.DatabaseDef.InMemory = Ptr(false)
	}

	if *dbConfig.DatabaseDef.InMemory {
		if dbConfig.DatabaseDef.Id == nil {
			mllog.Fatal("missing explicit Id for In-Memory db: ", dbConfig.ConfigFilePath)
		}
		dbConfig.DatabaseDef.Path = Ptr(":memory:")
	} else {
		if *dbConfig.DatabaseDef.Path == "" {
			mllog.Fatal("no path specified for db: ", dbConfig.ConfigFilePath)
		}

		// resolves '~'
		dbConfig.DatabaseDef.Path = Ptr(expandHomeDir(*dbConfig.DatabaseDef.Path, "database file"))
		if dbConfig.DatabaseDef.Id == nil {
			dbConfig.DatabaseDef.Id = Ptr(
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
	dbConfig.ToCreate = *dbConfig.DatabaseDef.InMemory || !fileExists(*dbConfig.DatabaseDef.Path)

	return dbConfig
}
