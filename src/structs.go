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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/iancoleman/orderedmap"
)

// This is the ws4sqlite error type

type wsError struct {
	RequestIdx int    `json:"reqIdx"`
	Msg        string `json:"error"`
	Code       int    `json:"-"`
}

func (m wsError) Error() string {
	return m.Msg
}

func newWSErrorf(reqIdx int, code int, msg string, elements ...interface{}) wsError {
	return wsError{reqIdx, fmt.Sprintf(msg, elements...), code}
}

func newWSError(reqIdx int, code int, msg string) wsError {
	return wsError{reqIdx, msg, code}
}

// These are for parsing the config file (from YAML)
// and storing additional context

type scheduledTask struct {
	Schedule       *string  `yaml:"schedule"`
	AtStartup      *bool    `yaml:"atStartup"`
	DoVacuum       bool     `yaml:"doVacuum"`
	DoBackup       bool     `yaml:"doBackup"`
	BackupTemplate string   `yaml:"backupTemplate"`
	NumFiles       int      `yaml:"numFiles"`
	Statements     []string `yaml:"statements"`
	Db             *db
}

type credentialsCfg struct {
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	HashedPassword string `yaml:"hashedPassword"`
}

type authr struct {
	Mode            string           `yaml:"mode"` // 'INLINE' or 'HTTP'
	CustomErrorCode *int             `yaml:"customErrorCode"`
	ByQuery         string           `yaml:"byQuery"`
	ByCredentials   []credentialsCfg `yaml:"byCredentials"`
	HashedCreds     map[string][]byte
}

type storedStatement struct {
	Id  string `yaml:"id"`
	Sql string `yaml:"sql"`
}

type db struct {
	Id                      string
	Path                    string
	CompanionFilePath       string
	Auth                    *authr            `yaml:"auth"`
	ReadOnly                bool              `yaml:"readOnly"`
	CORSOrigin              string            `yaml:"corsOrigin"`
	UseOnlyStoredStatements bool              `yaml:"useOnlyStoredStatements"`
	DisableWALMode          bool              `yaml:"disableWALMode"`
	Maintenance             *scheduledTask    `yaml:"maintenance"`
	ScheduledTasks          []scheduledTask   `yaml:"scheduledTasks"`
	StoredStatement         []storedStatement `yaml:"storedStatements"`
	InitStatements          []string          `yaml:"initStatements"`
	Db                      *sql.DB
	DbConn                  *sql.Conn
	StoredStatsMap          map[string]string
	Mutex                   *sync.Mutex
}

type config struct {
	Bindhost  string
	Port      int
	Databases []db
	ServeDir  *string
}

// These are for parsing the request (from JSON)

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type requestItem struct {
	Query       string            `json:"query"`
	Statement   string            `json:"statement"`
	NoFail      bool              `json:"noFail"`
	Values      json.RawMessage   `json:"values"`
	ValuesBatch []json.RawMessage `json:"valuesBatch"`
}

type request struct {
	ResultFormat *string       `json:"resultFormat"`
	Credentials  *credentials  `json:"credentials"`
	Transaction  []requestItem `json:"transaction"`
}

type requestParams struct {
	UnmarshalledDict  map[string]any
	UnmarshalledArray []any
}

// These are for generating the response

type responseItem struct {
	Success          bool                    `json:"success"`
	RowsUpdated      *int64                  `json:"rowsUpdated,omitempty"`
	RowsUpdatedBatch []int64                 `json:"rowsUpdatedBatch,omitempty"`
	ResultHeaders    []string                `json:"resultHeaders,omitempty"`
	ResultSet        []orderedmap.OrderedMap `json:"resultSet,omitnil"`     // omitnil is used by jettison
	ResultSetList    [][]interface{}         `json:"resultSetList,omitnil"` // omitnil is used by jettison
	Error            string                  `json:"error,omitempty"`
}

type response struct {
	Results []responseItem `json:"results"`
}
