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

package structs

import (
	"database/sql"
	"sync"
)

// These are for parsing the config file (from YAML)
// and storing additional context

type ScheduledTask struct {
	Schedule       *string  `yaml:"schedule"`
	AtStartup      *bool    `yaml:"atStartup"`
	DoVacuum       bool     `yaml:"doVacuum"`
	DoBackup       bool     `yaml:"doBackup"`
	BackupTemplate string   `yaml:"backupTemplate"`
	NumFiles       int      `yaml:"numFiles"`
	Statements     []string `yaml:"statements"`
	Db             *Db
}

type CredentialsCfg struct {
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	HashedPassword string `yaml:"hashedPassword"`
}

type Authr struct {
	Mode            string           `yaml:"mode"` // 'INLINE' or 'HTTP'
	CustomErrorCode *int             `yaml:"customErrorCode"`
	ByQuery         string           `yaml:"byQuery"`
	ByCredentials   []CredentialsCfg `yaml:"byCredentials"`
	HashedCreds     map[string][]byte
}

type StoredStatement struct {
	Id  string `yaml:"id"`
	Sql string `yaml:"sql"`
}

type DatabaseDef struct {
	Type           *string `yaml:"type"`           // SQLITE, DUCKDB (case insensitive)
	InMemory       *bool   `yaml:"inMemory"`       // if type = SQLITE | DUCKDB, default = false
	Path           *string `yaml:"path"`           // if type = SQLITE | DUCKDB and InMemory = false
	Id             *string `yaml:"id"`             // if type = SQLITE | DUCKDB, optional if InMemory = true
	DisableWALMode *bool   `yaml:"disableWALMode"` // if type = SQLITE
	ReadOnly       bool    `yaml:"readOnly"`
}

type Db struct {
	ConfigFilePath          string
	DatabaseDef             DatabaseDef       `yaml:"database"`
	Auth                    *Authr            `yaml:"auth"`
	CORSOrigin              string            `yaml:"corsOrigin"`
	UseOnlyStoredStatements bool              `yaml:"useOnlyStoredStatements"`
	Maintenance             *ScheduledTask    `yaml:"maintenance"`
	ScheduledTasks          []ScheduledTask   `yaml:"scheduledTasks"`
	StoredStatement         []StoredStatement `yaml:"storedStatements"`
	InitStatements          []string          `yaml:"initStatements"`
	ToCreate                bool              // if type = SQLITE
	ConnectionGetter        func() (*sql.DB, error)
	Db                      *sql.DB
	DbConn                  *sql.Conn
	StoredStatsMap          map[string]string
	Mutex                   *sync.Mutex
}

type Config struct {
	Bindhost  string
	Port      int
	Databases []Db
	ServeDir  *string
}
