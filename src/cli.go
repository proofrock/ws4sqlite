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
	"flag"
	"os"
	"strings"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
	"gopkg.in/yaml.v2"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, " ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, strings.TrimSpace(value))
	return nil
}

// This method parses the commandline and produces a config instance, either
// by filling in the database information when specifying a single database
// on the commandline, or loading a YAML config file.
//
// The config file must then be "completed" by verifying the coherence of the
// various fields, and generating the pointers to database, mutexes etc.
func parseCLI() structs.Config {
	// We don't use the "main" flag set because Parse() is not repeatable (for testing)
	fs := flag.NewFlagSet("ws4sql", flag.ExitOnError)

	// cli parameters
	var dbFiles arrayFlags
	fs.Var(&dbFiles, "db", "Repeatable; paths of config yamls")

	quickDb := fs.String("quick-db", "", "Shortcut to simply open a sqlite db file (for quicker config)")

	serveDir := fs.String("serve-dir", "", "A directory to serve with builtin HTTP server")

	bindHost := fs.String("bind-host", "0.0.0.0", "The host to bind")
	port := fs.Int("port", 12321, "Port for the web service")
	version := fs.Bool("version", false, "Display the version number")

	if err := fs.Parse(os.Args[1:]); err != nil {
		mllog.Fatalf("parsing commandline arguments: %s", err.Error())
	}

	// version is printed before calling this method, so nothing left to do but exit
	if *version {
		os.Exit(0)
	}

	var ret structs.Config

	if *quickDb != "" {
		if len(dbFiles) > 0 {
			mllog.Fatal("--quick-db must be the only database configured, if present")
		}

		ret.Databases = append(ret.Databases, structs.Db{
			ConfigFilePath: "quick db setting",
			DatabaseDef: structs.DatabaseDef{
				Type: utils.Ptr("SQLITE"),
				Path: quickDb,
			},
		})
	} else if len(dbFiles) > 0 {
		for i := range dbFiles {
			yamlFile := utils.ExpandHomeDir(dbFiles[i], "companion file")

			var dbConfig structs.Db
			dbConfig.ConfigFilePath = yamlFile

			if utils.FileExists(yamlFile) {
				cfgData, err := os.ReadFile(yamlFile)
				if err != nil {
					mllog.Fatal("in reading config file: ", err.Error())
				}

				if err = yaml.Unmarshal(cfgData, &dbConfig); err != nil {
					mllog.Fatal("in parsing config file: ", err.Error())
				}
			} else {
				mllog.Fatal("non-existing config file: ", yamlFile)
			}

			ret.Databases = append(ret.Databases, dbConfig)
		}
	} else if *serveDir == "" {
		mllog.Fatal("no database and no dir to serve specified")
	}

	if *serveDir != "" {
		sd := *serveDir
		// resolves '~'
		sd = utils.ExpandHomeDir(sd, "directory to serve")

		if !utils.DirExists(sd) {
			mllog.Fatalf("directory to serve does not exist: %s", *serveDir)
		}
		ret.ServeDir = &sd
	}

	// embed the cli parameters in the config
	ret.Bindhost = *bindHost
	ret.Port = *port

	return ret
}
