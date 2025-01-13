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
	"strings"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

type Engine interface {
	GetVersion() (string, error)
	GetDefaultIsolationLevel() sql.IsolationLevel
	CheckConfig(dbConfig structs.Db) structs.Db
	CheckRequest(body structs.Request) *structs.WsError
}

const ID_SQLITE = "SQLITE"
const ID_DUCKDB = "DUCKDB"

var FLAV_SQLITE Engine = &sqliteEngine{}
var FLAV_DUCKDB Engine = &duckdbEngine{}

// Checks the config passed and fails (logs & exits) if not valid.
// If valid, returns the normalized ID.
func NormalizeConf(declaredType *string) *string {
	if declaredType == nil {
		mllog.StdOutf("  + No type specified, assuming SQLITE")
		return utils.Ptr(ID_SQLITE)
	} else {
		engine := strings.ToUpper(*declaredType)
		if engine != ID_SQLITE && engine != ID_DUCKDB {
			mllog.Fatalf("invalid type: %s", *declaredType)
			return nil // not reachable
		}
		return utils.Ptr(engine)
	}
}

// Requires that the string is already normalized w/ the method above
func GetFlavorForStr(str string) Engine {
	if str == ID_SQLITE {
		return FLAV_SQLITE
	}
	return FLAV_DUCKDB
}

// Must be a function and not a struct field to avoid circular references :-(
func GetFlavorForDb(db structs.Db) Engine {
	return GetFlavorForStr(*db.DatabaseDef.Type)
}
