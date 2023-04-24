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
	"database/sql"
	"encoding/json"
	"github.com/mitchellh/go-homedir"
	mllog "github.com/proofrock/go-mylittlelogger"
	"os"
	"strings"
)

// Uppercases (?) the first letter of a string
func capitalize(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}

// Does a file exist? No error returned.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		mllog.Fatal("in stating file '", filename, "': ", err.Error())
	}
	return !info.IsDir()
}

// Does a dir exist? No error returned.
func dirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		mllog.Fatal("in stating dir '", dirname, "': ", err.Error())
	}
	return info.IsDir()
}

// Maps the raw JSON messages to a proper map, to manage unstructured JSON parsing;
// see https://noamt.medium.com/using-gos-json-rawmessage-a2371a1c11b7
func raw2vals(raw map[string]json.RawMessage) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	for key, rawVal := range raw {
		var val interface{}
		if err := json.Unmarshal(rawVal, &val); err != nil {
			return nil, err
		}
		ret[key] = val
	}
	return ret, nil
}

// Maps named parameters to the proper sql type, needed to use named params
func vals2nameds(vals map[string]interface{}) []interface{} {
	var nameds []interface{}
	for key, val := range vals {
		nameds = append(nameds, sql.Named(key, val))
	}
	return nameds
}

// Processes paths with home (tilde) expansion. Fails if not valid
func expandHomeDir(path string, desc string) string {
	ePath, err := homedir.Expand(path)
	if err != nil {
		mllog.Fatalf("in expanding %s path: %s", desc, err.Error())
	}
	return ePath
}

// Returns the first two components of a column-delimited string; if there's no column, second is ""
func splitOnColon(toSplit string) (string, string) {
	components := append(strings.SplitN(toSplit, ":", 2), "")
	return components[0], components[1]
}
