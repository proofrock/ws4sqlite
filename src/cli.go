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
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	mllog "github.com/proofrock/go-mylittlelogger"
	"gopkg.in/yaml.v2"
)

func parseCLI() config {
	cfgPath := flag.String("cfg", "", "Path of the YAML config file.")
	dbFilePath := flag.String("db", "", "Path of the database file.")
	bindHost := flag.String("bind-host", "0.0.0.0", "The host to bind (default: 0.0.0.0).")
	port := flag.Int("port", 12321, "Port for the web service (default: 12321).")
	version := flag.Bool("version", false, "Display the version number")

	flag.Parse()

	if *version {
		os.Exit(0)
	}

	var ret config
	var err error

	if (*cfgPath == "") == (*dbFilePath == "") {
		mllog.Fatal("one and only one of --cfg and --db must be specified")
	}

	if *cfgPath != "" {
		if !strings.HasSuffix(*cfgPath, ".yaml") {
			mllog.Fatal("config file must end with .yaml")
		}

		*cfgPath, err = homedir.Expand(*cfgPath)
		if err != nil {
			mllog.Fatal("in expanding config file path: ", err.Error())
		}

		if !fileExists(*cfgPath) {
			mllog.Fatalf("config file %s does not exist", *cfgPath)
		}

		cfgData, err := os.ReadFile(*cfgPath)
		if err != nil {
			mllog.Fatal("in reading config file: ", err.Error())
		}

		if err = yaml.Unmarshal(cfgData, &ret); err != nil {
			mllog.Fatal("in parsing config file: ", err.Error())
		}
	} else {
		if !strings.HasSuffix(*dbFilePath, ".db") {
			mllog.Fatal("database file must end with .db")
		}

		*dbFilePath, err = homedir.Expand(*dbFilePath)
		if err != nil {
			mllog.Fatal("in expanding database file path: ", err.Error())
		}

		dbFn := filepath.Base(*dbFilePath)

		ret = config{
			Databases: []db{
				{
					Id:   dbFn[0 : len(dbFn)-3],
					Path: *dbFilePath,
				},
			},
		}
	}

	ret.Bindhost = *bindHost
	ret.Port = *port

	return ret
}
