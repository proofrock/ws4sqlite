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
	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/engines"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

// In this file, a config passed by an user on the commandline is checked and
// if necessary normalized (default values, ecc).

func ckConfig(dbConfig structs.Db) structs.Db {
	mllog.StdOutf("- Parsing config file: %s", dbConfig.ConfigFilePath)

	dbConfig.DatabaseDef.Type = engines.NormalizeConf(dbConfig.DatabaseDef.Type)

	if dbConfig.DatabaseDef.InMemory == nil {
		dbConfig.DatabaseDef.InMemory = utils.Ptr(false)
	}

	ret := engines.GetFlavorForDb(dbConfig).CheckConfig(dbConfig)

	if ret.DatabaseDef.ReadOnly && dbConfig.ToCreate && len(dbConfig.InitStatements) > 0 {
		mllog.Fatal("a new db cannot be read only and have init statement: ", dbConfig.ConfigFilePath)
	}

	return ret
}
