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
	"fmt"
	"os"
	"testing"

	mllog "github.com/proofrock/go-mylittlelogger"
)

func assert(t *testing.T, condition bool, err ...interface{}) {
	if !condition {
		t.Error(fmt.Sprint(err...))
	}
}

func cliTest(argv ...string) (config, string) {
	args := os.Args
	os.Args = append([]string{"app"}, argv...)
	defer func() { os.Args = args }()
	err := ""
	orig := mllog.WhenFatal
	mllog.WhenFatal = func(msg string) {
		err = msg
	}
	defer func() { mllog.WhenFatal = orig }()
	cfg := parseCLI()
	return cfg, err
}

func TestCliEmpty(t *testing.T) {
	_, err := cliTest()
	assert(t, err != "", "succeeded, but shouldn't have ", err)
}

func TestCliMem(t *testing.T) {
	cfg, err := cliTest("--mem-db", "mem1")
	assert(t, err == "", "did not succeed ", err)
	assert(t, len(cfg.Databases) == 1, "only one db should be configured")
	assert(t, cfg.Databases[0].Id == "mem1", "the db has a wrong id")
	assert(t, cfg.Databases[0].Path == ":memory:", "the db is not on memory")
}

func TestCliFile(t *testing.T) {
	cfg, err := cliTest("--db", "../test/test.db")
	assert(t, err == "", "did not succeed ", err)
	assert(t, len(cfg.Databases) == 1, "only one db should be configured")
	assert(t, cfg.Databases[0].Id == "test", "the db has a wrong id")
	assert(t, cfg.Databases[0].Path == "../test/test.db", "the db has not the correct Path")
}

func TestCliMixed(t *testing.T) {
	cfg, err := cliTest("--db", "../test/test1.db", "--mem-db", "mem1", "--mem-db", "mem2", "--db", "../test/test2.db")
	assert(t, err == "", "did not succeed ", err)
	assert(t, len(cfg.Databases) == 4, "four db should be configured")
	assert(t, cfg.Databases[0].Id == "test1", "the db has a wrong id")
	assert(t, cfg.Databases[0].Path == "../test/test1.db", "the db has not the correct Path")
	assert(t, cfg.Databases[1].Id == "test2", "the db has a wrong id")
	assert(t, cfg.Databases[1].Path == "../test/test2.db", "the db has not the correct Path")
	assert(t, cfg.Databases[2].Id == "mem1", "the db has a wrong id")
	assert(t, cfg.Databases[2].Path == ":memory:", "the db is not on memory")
	assert(t, cfg.Databases[3].Id == "mem2", "the db has a wrong id")
	assert(t, cfg.Databases[3].Path == ":memory:", "the db is not on memory")
}

func TestConfigs(t *testing.T) {
	cfg, err := cliTest("--db", "../test/test1.db", "--mem-db", "mem1:../test/mem1.yaml", "--mem-db", "mem2", "--db", "../test/test2.db")
	assert(t, err == "", "did not succeed ", err)
	assert(t, len(cfg.Databases) == 4, "four db should be configured")

	assert(t, cfg.Databases[0].Id == "test1", "the db has a wrong id")
	assert(t, cfg.Databases[0].Path == "../test/test1.db", "the db has not the correct Path")
	assert(t, cfg.Databases[0].HasConfigFile, "the db is not correctly marked regarding having config file")
	assert(t, cfg.Databases[0].Auth.Mode == authModeHttp, "the db has not the correct Auth Mode")
	assert(t, cfg.Databases[0].Auth.ByQuery == "", "the db has ByQuery with a value")
	assert(t, len(cfg.Databases[0].Auth.ByCredentials) == 2, "the db has not the correct number of credentials")
	assert(t, cfg.Databases[0].Auth.ByCredentials[0].User == "myUser1", "the db has not the correct first user")
	assert(t, cfg.Databases[0].Auth.ByCredentials[0].Password == "myHotPassword", "the db has not the correct first password")
	assert(t, cfg.Databases[0].Auth.ByCredentials[0].HashedPassword == "", "the db has not the correct first hashed password")
	assert(t, cfg.Databases[0].Auth.ByCredentials[1].User == "myUser2", "the db has not the correct second user")
	assert(t, cfg.Databases[0].Auth.ByCredentials[1].Password == "", "the db has not the correct second password")
	assert(t, len(cfg.Databases[0].Auth.ByCredentials[1].HashedPassword) == 64, "the db has not the correct second hashed password")
	assert(t, cfg.Databases[0].DisableWALMode, "the db has not the correct WAL mode")
	assert(t, cfg.Databases[0].ReadOnly, "the db has not the correct ReadOnly value")
	assert(t, !cfg.Databases[0].UseOnlyStoredStatements, "the db has not the correct UseOnlyStoredStatements value")
	assert(t, cfg.Databases[0].CORSOrigin == "", "the db has not the correct CORSOrigin value")
	assert(t, cfg.Databases[0].Maintenance.Schedule == "0 0 * * *", "the db has not the correct Maintenance.Schedule value")
	assert(t, cfg.Databases[0].Maintenance.DoVacuum, "the db has not the correct Maintenance.DoVacuum value")
	assert(t, cfg.Databases[0].Maintenance.DoBackup, "the db has not the correct Maintenance.DoBackup value")
	assert(t, cfg.Databases[0].Maintenance.BackupTemplate == "~/first_%s.db", "the db has not the correct Maintenance.BackupTemplate value")
	assert(t, cfg.Databases[0].Maintenance.NumFiles == 3, "the db has not the correct Maintenance.NumFiles value")
	assert(t, len(cfg.Databases[0].InitStatements) == 0, "the db has not the correct number of init statements")
	assert(t, len(cfg.Databases[0].StoredStatement) == 0, "the db has not the correct number of stored statements")

	assert(t, cfg.Databases[1].Id == "test2", "the db has a wrong id")
	assert(t, cfg.Databases[1].Path == "../test/test2.db", "the db has not the correct Path")
	assert(t, !cfg.Databases[1].HasConfigFile, "the db is not correctly marked regarding having config file")
	assert(t, cfg.Databases[1].Auth == nil, "the db has not the correct Auth Mode")
	assert(t, !cfg.Databases[1].DisableWALMode, "the db has not the correct WAL mode")
	assert(t, !cfg.Databases[1].ReadOnly, "the db has not the correct ReadOnly value")
	assert(t, !cfg.Databases[1].UseOnlyStoredStatements, "the db has not the correct UseOnlyStoredStatements value")
	assert(t, cfg.Databases[1].CORSOrigin == "", "the db has not the correct CORSOrigin value")
	assert(t, cfg.Databases[1].Maintenance == nil, "the db has not the correct Maintenance value")
	assert(t, len(cfg.Databases[1].InitStatements) == 0, "the db has not the correct number of init statements")
	assert(t, len(cfg.Databases[1].StoredStatement) == 0, "the db has not the correct number of stored statements")

	assert(t, cfg.Databases[2].Id == "mem1", "the db has a wrong id")
	assert(t, cfg.Databases[2].Path == ":memory:", "the db is not on memory")
	assert(t, cfg.Databases[2].HasConfigFile, "the db is not correctly marked regarding having config file")
	assert(t, cfg.Databases[2].Auth.Mode == authModeInline, "the db has not the correct Auth Mode")
	assert(t, cfg.Databases[2].Auth.ByQuery != "", "the db has ByQuery without a value")
	assert(t, len(cfg.Databases[2].Auth.ByCredentials) == 0, "the db has not the correct number of credentials")
	assert(t, !cfg.Databases[2].DisableWALMode, "the db has not the correct WAL mode")
	assert(t, !cfg.Databases[2].ReadOnly, "the db has not the correct ReadOnly value")
	assert(t, cfg.Databases[2].UseOnlyStoredStatements, "the db has not the correct UseOnlyStoredStatements value")
	assert(t, cfg.Databases[2].CORSOrigin != "", "the db has not the correct CORSOrigin value")
	assert(t, cfg.Databases[2].Maintenance == nil, "the db has not the correct Maintenance value")
	assert(t, len(cfg.Databases[2].InitStatements) == 4, "the db has not the correct number of init statements")
	assert(t, len(cfg.Databases[2].StoredStatement) == 2, "the db has not the correct number of stored statements")

	assert(t, cfg.Databases[3].Id == "mem2", "the db has a wrong id")
	assert(t, cfg.Databases[3].Path == ":memory:", "the db is not on memory")
	assert(t, !cfg.Databases[3].HasConfigFile, "the db is not correctly marked regarding having config file")
	assert(t, cfg.Databases[3].Auth == nil, "the db has not the correct Auth Mode")
	assert(t, !cfg.Databases[3].DisableWALMode, "the db has not the correct WAL mode")
	assert(t, !cfg.Databases[3].ReadOnly, "the db has not the correct ReadOnly value")
	assert(t, !cfg.Databases[3].UseOnlyStoredStatements, "the db has not the correct UseOnlyStoredStatements value")
	assert(t, cfg.Databases[3].CORSOrigin == "", "the db has not the correct CORSOrigin value")
	assert(t, cfg.Databases[3].Maintenance == nil, "the db has not the correct Maintenance value")
	assert(t, len(cfg.Databases[3].InitStatements) == 0, "the db has not the correct number of init statements")
	assert(t, len(cfg.Databases[3].StoredStatement) == 0, "the db has not the correct number of stored statements")
}
