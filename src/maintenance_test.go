package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/robfig/cron/v3"
)

func cleanMaintFiles(cfg config) {
	for i := range cfg.Databases {
		os.Remove(cfg.Databases[i].Path)
		bkpDir, bkpFile := filepath.Dir(cfg.Databases[i].Maintenance.BackupTemplate),
			filepath.Base(cfg.Databases[i].Maintenance.BackupTemplate)
		list, _ := filepath.Glob(fmt.Sprintf(filepath.Join(bkpDir, bkpFile), bkpTimeGlob))
		for i := range list {
			os.Remove(list[i])
		}
	}
}

// called only by tests, so it fits better here
func stopMaint() {
	if schedulings > 0 {
		scheduler.Stop()
		schedulings = 0
	}
	scheduler = cron.New()
}

// Takes two minutes
func TestMaintenance(t *testing.T) {
	defer Shutdown()

	cfg := config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []db{
			{
				Id:             "test1",
				Path:           "../test/test1.db",
				DisableWALMode: true, // generate only ".db" files
				Maintenance: &maintenance{
					Schedule:       "* * * * *",
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test1_%s.db",
					NumFiles:       1,
				},
			}, {
				Id:             "test2",
				Path:           "../test/test2.db",
				DisableWALMode: true, // generate only ".db" files
				Maintenance: &maintenance{
					Schedule:       "* * * * *",
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test2_%s.db",
					NumFiles:       1,
				},
			},
		},
	}

	cleanMaintFiles(cfg)
	defer cleanMaintFiles(cfg)

	go launch(cfg, true)

	time.Sleep(time.Second)

	if !fileExists(cfg.Databases[0].Path) || !fileExists(cfg.Databases[1].Path) {
		t.Error("db file not created")
		return
	}

	req := request{
		Transaction: []requestItem{
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

	if !fileExists(bk1) || !fileExists(bk2) {
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

	if !fileExists(bk3) || !fileExists(bk4) {
		t.Error("backup file not created, the second time")
		return
	}

	if fileExists(bk1) || fileExists(bk2) {
		t.Error("backup file not rotated")
		return
	}

	time.Sleep(time.Second)
}

func TestMaintWithReadOnly(t *testing.T) {
	defer os.Remove("../test/test.db")
	defer Shutdown()
	success := true

	oldForFatal := mllog.ForFatal
	mllog.ForFatal = func() { success = false }
	defer func() { mllog.ForFatal = oldForFatal }()

	cfg := config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []db{
			{
				Id:             "test",
				Path:           "../test/test.db",
				DisableWALMode: true, // generate only ".db" files
				ReadOnly:       true,
				Maintenance: &maintenance{
					Schedule:       "* * * * *",
					DoVacuum:       false,
					DoBackup:       true,
					BackupTemplate: "../test/test1_%s.db",
					NumFiles:       1,
				},
			},
		},
	}

	go launch(cfg, true)

	time.Sleep(time.Second)

	if success {
		t.Error("did succeed, but should not have")
	}
}
