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
	"fmt"
	"os"
	"testing"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
)

func assert(t *testing.T, condition bool, err ...interface{}) {
	if !condition {
		t.Error(fmt.Sprint(err...))
	}
}

func cliTest(argv ...string) (structs.Config, string) {
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

func TestQuickDb(t *testing.T) {
	cfg, err := cliTest("--quick-db", "../test/test1.db")
	assert(t, err == "", "did not succeed ", err)
	assert(t, len(cfg.Databases) == 1, "1 db should be configured")

	cfgdb := ckConfig(cfg.Databases[0])
	assert(t, *cfgdb.DatabaseDef.Id == "test1", "the db has a wrong id")
	assert(t, *cfgdb.DatabaseDef.Path == "../test/test1.db", "the db has not the correct Path")
	assert(t, cfgdb.ConfigFilePath != "", "the db is not correctly marked regarding having config file")
	assert(t, !*cfgdb.DatabaseDef.DisableWALMode, "the db has not the correct WAL mode")
	assert(t, !cfgdb.DatabaseDef.ReadOnly, "the db has not the correct ReadOnly value")
}

func TestCliEmpty(t *testing.T) {
	_, err := cliTest()
	assert(t, err != "", "succeeded, but shouldn't have ", err)
}

func TestCliOnlyServeDir(t *testing.T) {
	_, err := cliTest("--serve-dir", "../test")
	assert(t, err == "", "didn't succeed, but should have ", err)
}

func TestCliServeInvalidDir(t *testing.T) {
	_, err := cliTest("--serve-dir", "../test_non_existent")
	assert(t, err != "", "succeeded, but shouldn't have ", err)
}

func TestCliServeInvalidDir2(t *testing.T) {
	_, err := cliTest("--serve-dir", "../test/mem1.yaml") // it's a file
	assert(t, err != "", "succeeded, but shouldn't have ", err)
}

func TestConfigs(t *testing.T) {
	cfg, err := cliTest("--db", "../test/test1.yaml", "--db", "../test/mem1.yaml")
	assert(t, err == "", "did not succeed ", err)
	assert(t, len(cfg.Databases) == 2, "two db should be configured")

	cfgdb := ckConfig(cfg.Databases[0])
	assert(t, *cfgdb.DatabaseDef.Id == "test1", "the db has a wrong id")
	assert(t, *cfgdb.DatabaseDef.Path == "../test/test1.db", "the db has not the correct Path")
	assert(t, cfgdb.ConfigFilePath != "", "the db is not correctly marked regarding having config file")
	assert(t, cfgdb.Auth.Mode == authModeHttp, "the db has not the correct Auth Mode")
	assert(t, cfgdb.Auth.ByQuery == "", "the db has ByQuery with a value")
	assert(t, len(cfgdb.Auth.ByCredentials) == 2, "the db has not the correct number of credentials")
	assert(t, cfgdb.Auth.ByCredentials[0].User == "myUser1", "the db has not the correct first user")
	assert(t, cfgdb.Auth.ByCredentials[0].Password == "myHotPassword", "the db has not the correct first password")
	assert(t, cfgdb.Auth.ByCredentials[0].HashedPassword == "", "the db has not the correct first hashed password")
	assert(t, cfgdb.Auth.ByCredentials[1].User == "myUser2", "the db has not the correct second user")
	assert(t, cfgdb.Auth.ByCredentials[1].Password == "", "the db has not the correct second password")
	assert(t, len(cfgdb.Auth.ByCredentials[1].HashedPassword) == 64, "the db has not the correct second hashed password")
	assert(t, *cfgdb.DatabaseDef.DisableWALMode, "the db has not the correct WAL mode")
	assert(t, cfgdb.DatabaseDef.ReadOnly, "the db has not the correct ReadOnly value")
	assert(t, !cfgdb.UseOnlyStoredStatements, "the db has not the correct UseOnlyStoredStatements value")
	assert(t, cfgdb.CORSOrigin == "", "the db has not the correct CORSOrigin value")
	assert(t, *cfgdb.Maintenance.Schedule == "0 0 * * *", "the db has not the correct Maintenance.Schedule value")
	assert(t, cfgdb.Maintenance.DoVacuum, "the db has not the correct Maintenance.DoVacuum value")
	assert(t, cfgdb.Maintenance.DoBackup, "the db has not the correct Maintenance.DoBackup value")
	assert(t, cfgdb.Maintenance.BackupTemplate == "~/first_%s.db", "the db has not the correct Maintenance.BackupTemplate value")
	assert(t, cfgdb.Maintenance.NumFiles == 3, "the db has not the correct Maintenance.NumFiles value")
	assert(t, len(cfgdb.InitStatements) == 0, "the db has not the correct number of init statements")
	assert(t, len(cfgdb.StoredStatement) == 0, "the db has not the correct number of stored statements")

	cfgdb = ckConfig(cfg.Databases[1])
	assert(t, *cfgdb.DatabaseDef.Id == "mem1", "the db has a wrong id")
	assert(t, *cfgdb.DatabaseDef.Path == ":memory:", "the db is not on memory")
	assert(t, cfgdb.ConfigFilePath != "", "the db is not correctly marked regarding having config file")
	assert(t, cfgdb.Auth.Mode == authModeInline, "the db has not the correct Auth Mode")
	assert(t, cfgdb.Auth.ByQuery != "", "the db has ByQuery without a value")
	assert(t, len(cfgdb.Auth.ByCredentials) == 0, "the db has not the correct number of credentials")
	assert(t, !*cfgdb.DatabaseDef.DisableWALMode, "the db has not the correct WAL mode")
	assert(t, !cfgdb.DatabaseDef.ReadOnly, "the db has not the correct ReadOnly value")
	assert(t, cfgdb.UseOnlyStoredStatements, "the db has not the correct UseOnlyStoredStatements value")
	assert(t, cfgdb.CORSOrigin != "", "the db has not the correct CORSOrigin value")
	assert(t, cfgdb.Maintenance == nil, "the db has not the correct Maintenance value")
	assert(t, len(cfgdb.InitStatements) == 4, "the db has not the correct number of init statements")
	assert(t, len(cfgdb.StoredStatement) == 2, "the db has not the correct number of stored statements")
}
