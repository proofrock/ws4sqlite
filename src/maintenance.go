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
	"context"
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

// Used when deleting older backup files, the date/time is substituted with '?'
var bkpTimeGlob = strings.Repeat("?", len(bkpTimeFormat))

// Parses a backup plan, checks that it is well-formed and returns a function that
// will be called by cron and executes the plan.
func doMaint(db db) func() {
	var bkpDir, bkpFile string
	if db.Maintenance.DoBackup {
		var err error
		if db.Maintenance.BackupTemplate == "" {
			mllog.Fatal("the backup template must have a value")
		}

		db.Maintenance.BackupTemplate, err = homedir.Expand(db.Maintenance.BackupTemplate)
		if err != nil {
			mllog.Fatal("in expanding bkp template path: ", err.Error())
		}

		bkpDir, bkpFile = filepath.Dir(db.Maintenance.BackupTemplate), filepath.Base(db.Maintenance.BackupTemplate)

		if !strings.Contains(bkpFile, "%s") || strings.Count(bkpFile, "%") != 1 {
			mllog.Fatalf("the backup file name must contain a single '%%s' and no other '%%'")
		}
		if strings.Contains(bkpDir, "%") {
			mllog.Fatalf("the backup file dir must not contain a '%%'")
		}
		if _, err := os.Stat(bkpDir); errors.Is(err, os.ErrNotExist) {
			mllog.Fatal("the backup directory must exist")
		}

		if db.Maintenance.NumFiles < 1 {
			mllog.Fatal("the number of backup files to keep must be at least 1")
		}
	}

	// Execute a maintenance cycle, according to the plan. If so configured, does
	// a VACUUM, then a backup (with VACUUM INTO). Being a lambda, inherits the
	// plan from the parsing (above)
	//
	// Just log Errors when it fails. Doesn't of course block/abort anything.
	return func() {
		// Execute non-concurrently
		db.Mutex.Lock()
		defer db.Mutex.Unlock()

		if db.Maintenance.DoVacuum {
			if _, err := db.DbConn.ExecContext(context.Background(), "VACUUM"); err != nil {
				mllog.Error("maint (vacuum): ", err.Error())
				return
			}
		}

		if db.Maintenance.DoBackup {
			now := time.Now().Format(bkpTimeFormat)
			fname := fmt.Sprintf(filepath.Join(bkpDir, bkpFile), now)
			stat, err := db.DbConn.PrepareContext(context.Background(), "VACUUM INTO ?")
			if err != nil {
				mllog.Error("maint (backup prep): ", err.Error())
				return
			}
			defer stat.Close()
			if _, err := stat.Exec(fname); err != nil {
				mllog.Error("maint (backup): ", err.Error())
				return
			}
			// delete the backup files, except for the last n
			list, err := filepath.Glob(fmt.Sprintf(filepath.Join(bkpDir, bkpFile), bkpTimeGlob))
			if err != nil {
				mllog.Error("maint (pruning): ", err.Error())
				return
			}
			sort.Strings(list)
			for i := 0; i < len(list)-db.Maintenance.NumFiles; i++ {
				os.Remove(list[i])
			}
		}

		if len(db.Maintenance.Statements) > 0 {
			for idx := range db.Maintenance.Statements {
				if _, err := db.DbConn.ExecContext(context.Background(), db.Maintenance.Statements[idx]); err != nil {
					mllog.Errorf("maint (statement #%d): %s", idx, err.Error())
				}
			}
		}
	}
}

var scheduler = cron.New()
var schedulings = 0
var exprDesc, _ = cronDesc.NewDescriptor()

// Calls the parsing of the maintenance plan config, via doMaint(), and adds the
// resulting maintenance func to be executed by cron
func parseMaint(db *db) {
	// is there at least one btw schedule and atStartup?
	isOk := false
	if db.Maintenance.Schedule != nil {
		if _, err := scheduler.AddFunc(*db.Maintenance.Schedule, doMaint(*db)); err != nil {
			mllog.Fatal(err.Error())
		}
		schedulings++
		// Also prints a log containing the human-readable translation of the cron schedule
		if descr, err := exprDesc.ToDescription(*db.Maintenance.Schedule, cronDesc.Locale_en); err != nil {
			mllog.Fatal("error in decoding schedule: ", err.Error())
		} else {
			mllog.StdOut("  + Maintenance scheduled ", strings.ToLower(descr))
		}
		isOk = true
	}
	if db.Maintenance.AtStartup != nil && *db.Maintenance.AtStartup {
		mllog.StdOut("  + Maintenance at startup")
		isOk = true
	}
	if !isOk {
		mllog.Fatal("error: maintenance must be scheduled or atStartup")
	}
}

// Called by the launch function to actually start the cron engine.
// Does it only if there's something to do.
func startMaint(map[string]db) {
	for id, _ := range dbs {
		db := dbs[id]
		if db.Maintenance != nil && db.Maintenance.AtStartup != nil && *db.Maintenance.AtStartup {
			doMaint(db)()
		}
	}
	if schedulings > 0 {
		scheduler.Start()
	}
}
