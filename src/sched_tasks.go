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
	"github.com/proofrock/ws4sql/structs"

	cronDesc "github.com/lnquy/cron"
	"github.com/robfig/cron/v3"
)

const bkpTimeFormat = "060102-1504"

// Used when deleting older backup files, the date/time is substituted with '?'
var bkpTimeGlob = strings.Repeat("?", len(bkpTimeFormat))

// Parses a backup plan, checks that it is well-formed and returns a function that
// will be called by cron and executes the plan.
func doTask(task structs.ScheduledTask) func() {
	var bkpDir, bkpFile string
	if task.DoBackup {
		var err error
		if task.BackupTemplate == "" {
			mllog.Fatal("the backup template must have a value")
		}

		task.BackupTemplate, err = homedir.Expand(task.BackupTemplate)
		if err != nil {
			mllog.Fatal("in expanding bkp template path: ", err.Error())
		}

		bkpDir, bkpFile = filepath.Dir(task.BackupTemplate), filepath.Base(task.BackupTemplate)

		if !strings.Contains(bkpFile, "%s") || strings.Count(bkpFile, "%") != 1 {
			mllog.Fatalf("the backup file name must contain a single '%%s' and no other '%%'")
		}
		if strings.Contains(bkpDir, "%") {
			mllog.Fatalf("the backup file dir must not contain a '%%'")
		}
		if _, err := os.Stat(bkpDir); errors.Is(err, os.ErrNotExist) {
			mllog.Fatal("the backup directory must exist")
		}

		if task.NumFiles < 1 {
			mllog.Fatal("the number of backup files to keep must be at least 1")
		}
	}

	// Execute a task, according to the plan. If so configured, does
	// a VACUUM, then a backup (with VACUUM INTO). Being a lambda, inherits the
	// plan from the parsing (above)
	//
	// Just log Errors when it fails. Doesn't of course block/abort anything.
	return func() {
		// Execute non-concurrently
		task.Db.Mutex.Lock()
		defer task.Db.Mutex.Unlock()

		if task.DoVacuum {
			if _, err := task.Db.DbConn.ExecContext(context.Background(), "VACUUM"); err != nil {
				mllog.Error("sched. task (vacuum): ", err.Error())
				return
			}
		}

		if task.DoBackup {
			now := time.Now().Format(bkpTimeFormat)
			fname := fmt.Sprintf(filepath.Join(bkpDir, bkpFile), now)
			stat, err := task.Db.DbConn.PrepareContext(context.Background(), "VACUUM INTO ?")
			if err != nil {
				mllog.Error("sched. task (backup prep): ", err.Error())
				return
			}
			defer stat.Close()
			if _, err := stat.Exec(fname); err != nil {
				mllog.Error("sched. task (backup): ", err.Error())
				return
			}
			// delete the backup files, except for the last n
			list, err := filepath.Glob(fmt.Sprintf(filepath.Join(bkpDir, bkpFile), bkpTimeGlob))
			if err != nil {
				mllog.Error("sched. task (pruning bkp files): ", err.Error())
				return
			}
			sort.Strings(list)
			for i := 0; i < len(list)-task.NumFiles; i++ {
				os.Remove(list[i])
			}
		}

		if len(task.Statements) > 0 {
			for idx := range task.Statements {
				if _, err := task.Db.DbConn.ExecContext(context.Background(), task.Statements[idx]); err != nil {
					mllog.Errorf("sched. task (statement #%d): %s", idx, err.Error())
				}
			}
		}
	}
}

var scheduler = cron.New()
var haySchedules = false
var startupTasks = []structs.ScheduledTask{}
var exprDesc, _ = cronDesc.NewDescriptor()

// Calls the parsing of the scheduled tasks config, via doTask(), and adds the
// resulting task to be executed by cron
func parseTasks(db *structs.Db) {
	for idx := range db.ScheduledTasks {
		db.ScheduledTasks[idx].Db = db // back reference
		// is there at least one btw schedule and atStartup?
		isOk := false
		if db.ScheduledTasks[idx].Schedule != nil {
			if _, err := scheduler.AddFunc(*db.ScheduledTasks[idx].Schedule, doTask(db.ScheduledTasks[idx])); err != nil {
				mllog.Fatal(err.Error())
			}
			haySchedules = true
			// Also prints a log containing the human-readable translation of the cron schedule
			if descr, err := exprDesc.ToDescription(*db.ScheduledTasks[idx].Schedule, cronDesc.Locale_en); err != nil {
				mllog.Fatal("error in decoding schedule: ", err.Error())
			} else {
				mllog.StdOutf("  + Task %d scheduled %s", idx, strings.ToLower(descr))
			}
			isOk = true
		}
		if db.ScheduledTasks[idx].AtStartup != nil && *db.ScheduledTasks[idx].AtStartup {
			mllog.StdOutf("  + Task %d scheduled at startup", idx)
			startupTasks = append(startupTasks, db.ScheduledTasks[idx])
			isOk = true
		}
		if !isOk {
			mllog.Fatalf("error: task %d must be scheduled or atStartup", idx)
		}
	}
}

// Called by the launch function to execute the startup tasks and start the cron engine.
// Does it only if there's something to do.
func startTasks() {
	for idx := range startupTasks {
		doTask(startupTasks[idx])()
	}
	if haySchedules {
		scheduler.Start()
	}
}
