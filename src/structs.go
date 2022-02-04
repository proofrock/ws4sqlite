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
	"encoding/json"
	"fmt"
	"sync"
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

func newWSError(reqIdx int, code int, msg string, elements ...interface{}) wsError {
	return wsError{reqIdx, fmt.Sprintf(msg, elements...), code}
}

// These are for parsing the config file (from YAML)
// and storing additional context

type maintenance struct {
	Schedule       string `yaml:"schedule"`
	DoVacuum       bool   `yaml:"doVacuum"`
	DoBackup       bool   `yaml:"doBackup"`
	BackupTemplate string `yaml:"backupTemplate"`
	NumFiles       int    `yaml:"numFiles"`
}

type credentialsCfg struct {
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	HashedPassword string `yaml:"hashedPassword"`
}

type authr struct {
	Mode          string           `yaml:"mode"` // 'INLINE' or 'HTTP'
	ByQuery       string           `yaml:"byQuery"`
	ByCredentials []credentialsCfg `yaml:"byCredentials"`
	HashedCreds   map[string][]byte
}

type storedStatement struct {
	Id  string `yaml:"id"`
	Sql string `yaml:"sql"`
}

type db struct {
	Id                      string            `yaml:"id"`
	Path                    string            `yaml:"path"`
	Auth                    *authr            `yaml:"auth"`
	ReadOnly                bool              `yaml:"readOnly"`
	CORSOrigin              string            `yaml:"corsOrigin"`
	UseOnlyStoredStatements bool              `yaml:"useOnlyStoredStatements"`
	DisableWALMode          bool              `yaml:"disableWALMode"`
	Maintenance             *maintenance      `yaml:"maintenance"`
	StoredStatement         []storedStatement `yaml:"storedStatements"`
	InitStatements          []string          `yaml:"initStatements"`
	Db                      *sql.DB           `yaml:"-"`
	StoredStatsMap          map[string]string `yaml:"-"`
	Mutex                   *sync.Mutex       `yaml:"-"`
}

type config struct {
	Bindhost  string `yaml:"-"`
	Port      int    `yaml:"-"`
	Databases []db   `yaml:"databases"`
}

// These are for parsing the request (from JSON)

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type requestItemCrypto struct {
	Password         string   `json:"password"`
	Fields           []string `json:"fields"`
	CompressionLevel int      `json:"compressionLevel"`
}

type requestItem struct {
	Query       string                       `json:"query"`
	Statement   string                       `json:"statement"`
	NoFail      bool                         `json:"noFail"`
	Values      map[string]json.RawMessage   `json:"values"`
	ValuesBatch []map[string]json.RawMessage `json:"valuesBatch"`
	Encoder     *requestItemCrypto           `json:"encoder"`
	Decoder     *requestItemCrypto           `json:"decoder"`
}

type request struct {
	Credentials *credentials  `json:"credentials"`
	Transaction []requestItem `json:"transaction"`
}

// These are for generating the response

type responseItem struct {
	Success          bool                     `json:"success"`
	RowsUpdated      *int64                   `json:"rowsUpdated,omitempty"`
	RowsUpdatedBatch []int64                  `json:"rowsUpdatedBatch,omitempty"`
	ResultSet        []map[string]interface{} `json:"resultSet,omitnil"` // omitnil is used by jettison
	Error            string                   `json:"error,omitempty"`
}

func ResItemEmpty() responseItem {
	return responseItem{}
}

func ResItem4Query(resultSet []map[string]interface{}) responseItem {
	return responseItem{true, nil, nil, resultSet, ""}
}

func ResItem4Statement(rowsUpdated int64) responseItem {
	return responseItem{true, &rowsUpdated, nil, nil, ""}
}

func ResItem4Batch(rowsUpdatedBatch []int64) responseItem {
	return responseItem{true, nil, rowsUpdatedBatch, nil, ""}
}

func ResItem4Error(error string) responseItem {
	return responseItem{false, nil, nil, nil, error}
}

type response struct {
	Results []responseItem `json:"results"`
}
