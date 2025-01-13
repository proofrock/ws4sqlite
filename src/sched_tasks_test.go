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
	"path/filepath"
	"testing"
	"time"

	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
	"github.com/robfig/cron/v3"
)

// The post-0.14 "scheduledTasks" structure is only actually tested in TestAtStartupMultiple, but the contents of the
// "maintenance" structure are copied to it anyway so the old tests for "maintenance" should suffice for the new one too

func cleanSchedTasksFiles(cfg structs.Config) {
	for i := range cfg.Databases {
		os.Remove(*cfg.Databases[i].DatabaseDef.Path)
		bkpDir, bkpFile := filepath.Dir(cfg.Databases[i].Maintenance.BackupTemplate),
			filepath.Base(cfg.Databases[i].Maintenance.BackupTemplate)
		list, _ := filepath.Glob(fmt.Sprintf(filepath.Join(bkpDir, bkpFile), bkpTimeGlob))
		for i := range list {
			os.Remove(list[i])
		}
	}
}

// called only by tests, so it fits better here
func stopScheduler() {
	if haySchedules {
		scheduler.Stop()
		haySchedules = false
	}
	scheduler = cron.New()
}

// Takes two minutes
func TestSchedTasks(t *testing.T) {
	defer os.Remove("../test/test1.db")
	defer os.Remove("../test/test2.db")
	defer Shutdown()

	sched := "* * * * *"

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test1"),
					Path:           utils.Ptr("../test/test1.db"),
					DisableWALMode: utils.Ptr(true), // generate only ".db" files
				},
				Maintenance: &structs.ScheduledTask{
					Schedule:       &sched,
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test1_%s.db",
					NumFiles:       1,
				},
			}, {
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test2"),
					Path:           utils.Ptr("../test/test2.db"),
					DisableWALMode: utils.Ptr(true), // generate only ".db" files
				},
				Maintenance: &structs.ScheduledTask{
					Schedule:       &sched,
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test2_%s.db",
					NumFiles:       1,
				},
			},
		},
	}

	cleanSchedTasksFiles(cfg)
	defer cleanSchedTasksFiles(cfg)

	go launch(cfg, true)

	time.Sleep(time.Second)

	if !utils.FileExists(*cfg.Databases[0].DatabaseDef.Path) || !utils.FileExists(*cfg.Databases[1].DatabaseDef.Path) {
		t.Error("db file not created")
		return
	}

	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
			},
		},
	}
	code, _, _ := call("test1", req, t)
	if code != 200 {
		t.Error("did not succeed")
		return
	}

	time.Sleep(time.Minute)

	now := time.Now().Format(bkpTimeFormat)
	bk1 := fmt.Sprintf(cfg.Databases[0].Maintenance.BackupTemplate, now)
	bk2 := fmt.Sprintf(cfg.Databases[1].Maintenance.BackupTemplate, now)

	if !utils.FileExists(bk1) || !utils.FileExists(bk2) {
		t.Error("backup file not created")
		return
	}

	stat1, _ := os.Stat(bk1)
	stat2, _ := os.Stat(bk2)

	if stat2.Size() >= stat1.Size() {
		t.Error("backup files sizes are inconsistent")
	}

	time.Sleep(time.Minute)

	now = time.Now().Format(bkpTimeFormat)
	bk3 := fmt.Sprintf(cfg.Databases[0].Maintenance.BackupTemplate, now)
	bk4 := fmt.Sprintf(cfg.Databases[1].Maintenance.BackupTemplate, now)

	if !utils.FileExists(bk3) || !utils.FileExists(bk4) {
		t.Error("backup file not created, the second time")
		return
	}

	if utils.FileExists(bk1) || utils.FileExists(bk2) {
		t.Error("backup file not rotated")
		return
	}

	time.Sleep(time.Second)
}

// Takes one minute
func TestSchedTasksWithReadOnly(t *testing.T) {
	defer os.Remove("../test/test.db")
	defer Shutdown()

	sched := "* * * * *"

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test"),
					Path:           utils.Ptr("../test/test.db"),
					DisableWALMode: utils.Ptr(true), // generate only ".db" files
					ReadOnly:       true,
				},
				Maintenance: &structs.ScheduledTask{
					Schedule:       &sched,
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test_%s.db",
					NumFiles:       1,
				},
			},
		},
	}

	cleanSchedTasksFiles(cfg)
	defer cleanSchedTasksFiles(cfg)

	go launch(cfg, true)

	time.Sleep(time.Minute)

	now := time.Now().Format(bkpTimeFormat)
	bk1 := fmt.Sprintf(cfg.Databases[0].Maintenance.BackupTemplate, now)

	if !utils.FileExists(bk1) {
		t.Error("backup file not created")
		return
	}
}

// Takes one minute
func TestSchedTasksWithStatement(t *testing.T) {
	defer os.Remove("../test/test.db")
	defer Shutdown()

	sched := "* * * * *"

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test"),
					Path:           utils.Ptr("../test/test.db"),
					DisableWALMode: utils.Ptr(true), // generate only ".db" files
				},
				Maintenance: &structs.ScheduledTask{
					Schedule:   &sched,
					DoVacuum:   false,
					DoBackup:   false,
					Statements: []string{"INSERT INTO tbl VALUES (17)"},
				},
				InitStatements: []string{"CREATE TABLE tbl (num INTEGER)"},
			},
		},
	}

	go launch(cfg, true)

	time.Sleep(time.Minute)

	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT num FROM tbl",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed, but should have")
	}

	if fmt.Sprint(utils.GetDefault[float64](res.Results[0].ResultSet[0], "num")) != "17" {
		t.Error("scheduled statement probably didn't execute")
	}
}

func TestAtStartup(t *testing.T) {
	defer os.Remove("../test/test.db")
	defer Shutdown()

	t_r_u_e := true

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test"),
					Path:           utils.Ptr("../test/test.db"),
					DisableWALMode: utils.Ptr(true), // generate only ".db" files
				},
				Maintenance: &structs.ScheduledTask{
					AtStartup:      &t_r_u_e,
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test_%s.db",
					NumFiles:       1,
				},
			},
		},
	}

	cleanSchedTasksFiles(cfg)
	defer cleanSchedTasksFiles(cfg)

	go launch(cfg, true)
	now := time.Now().Format(bkpTimeFormat)

	time.Sleep(3 * time.Second)

	bk1 := fmt.Sprintf(cfg.Databases[0].Maintenance.BackupTemplate, now)

	if !utils.FileExists(bk1) {
		t.Error("backup file not created")
		return
	}
}

func TestAtStartupMultiple(t *testing.T) {
	defer os.Remove("../test/test.db")
	defer Shutdown()

	t_r_u_e := true

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test"),
					Path:           utils.Ptr("../test/test.db"),
					DisableWALMode: utils.Ptr(true), // generate only ".db" files
				},
				InitStatements: []string{
					"CREATE TABLE TMP (ID INTEGER)",
				},
				ScheduledTasks: []structs.ScheduledTask{
					{
						AtStartup:  &t_r_u_e,
						DoVacuum:   false,
						DoBackup:   false,
						Statements: []string{"INSERT INTO TMP VALUES (1)"},
					}, {
						AtStartup:  &t_r_u_e,
						DoVacuum:   false,
						DoBackup:   false,
						Statements: []string{"INSERT INTO TMP VALUES (2)"},
					},
				},
			},
		},
	}

	go launch(cfg, true)

	time.Sleep(3 * time.Second)

	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT ID AS CNT FROM TMP",
			},
		},
	}

	code, body, res := call("test", req, t)

	if code != 200 {
		t.Errorf("did not succeed (%d): %s", code, body)
		return
	}

	if len(res.Results[0].ResultSet) != 2 {
		t.Errorf("did not succeed -> %v", res.Results[0].ResultSet)
	}
}
