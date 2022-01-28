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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	mllog "github.com/proofrock/go-mylittlelogger"

	cronDesc "github.com/lnquy/cron"
	"github.com/robfig/cron/v3"
)

const bkpTimeFormat = "060102-1504"

var bkpTimeGlob = strings.Repeat("?", len(bkpTimeFormat))

func doMaint(id string, mntCfg maintenance, db *sql.DB) func() {
	mllog.Warn("Parsing for db ", id)
	var bkpDir, bkpFile string
	if mntCfg.DoBackup {
		var err error
		if mntCfg.BackupTemplate == "" {
			mllog.Fatal("the backup template must have a value")
		}

		mntCfg.BackupTemplate, err = homedir.Expand(mntCfg.BackupTemplate)
		if err != nil {
			mllog.Fatal("in expanding bkp template path: ", err.Error())
		}

		bkpDir, bkpFile = filepath.Dir(mntCfg.BackupTemplate), filepath.Base(mntCfg.BackupTemplate)

		if !strings.Contains(bkpFile, "%s") || strings.Count(bkpFile, "%") != 1 {
			mllog.Fatalf("the backup file name must contain a single '%%s' and no other '%%'")
		}
		if strings.Contains(bkpDir, "%") {
			mllog.Fatalf("the backup file dir must not contain a '%%'")
		}
		if _, err := os.Stat(bkpDir); errors.Is(err, os.ErrNotExist) {
			mllog.Fatal("the backup directory must exist")
		}

		if mntCfg.NumFiles < 1 {
			mllog.Fatal("the number of backup files to keep must be at least 1")
		}
	}
	return func() {
		if mntCfg.DoVacuum {
			if _, err := db.Exec("VACUUM"); err != nil {
				mllog.Error("maint (vacuum): ", err.Error())
				return
			}
		}
		if mntCfg.DoBackup {
			now := time.Now().Format(bkpTimeFormat)
			fname := fmt.Sprintf(filepath.Join(bkpDir, bkpFile), now)
			stat, err := db.Prepare("VACUUM INTO ?")
			if err != nil {
				mllog.Error("maint (backup prep): ", err.Error())
				return
			}
			defer stat.Close()
			if _, err := stat.Exec(fname); err != nil {
				mllog.Error("maint (backup): ", err.Error())
				return
			}
			// delete the backup files but for the last n
			list, err := filepath.Glob(fmt.Sprintf(filepath.Join(bkpDir, bkpFile), bkpTimeGlob))
			if err != nil {
				mllog.Error("maint (pruning): ", err.Error())
				return
			}
			sort.Strings(list)
			for i := 0; i < len(list)-mntCfg.NumFiles; i++ {
				os.Remove(list[i])
			}
		}
	}
}

var scheduler = cron.New()
var schedulings = 0
var exprDesc, _ = cronDesc.NewDescriptor()

func parseMaint(db *db) {
	if _, err := scheduler.AddFunc(db.Maintenance.Schedule, doMaint(db.Id, *db.Maintenance, db.Db)); err != nil {
		mllog.Fatal(err.Error())
	}
	schedulings++
	if descr, err := exprDesc.ToDescription(db.Maintenance.Schedule, cronDesc.Locale_en); err != nil {
		mllog.Fatal("error in decoding schedule: ", err.Error())
	} else {
		mllog.StdOut("  + Maintenance scheduled ", strings.ToLower(descr))
	}
}

func startMaint() {
	if schedulings > 0 {
		scheduler.Start()
	}
}
