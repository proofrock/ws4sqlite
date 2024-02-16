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
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/iancoleman/orderedmap"
	"github.com/mitchellh/go-homedir"
	mllog "github.com/proofrock/go-mylittlelogger"
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

// Maps named parameters to the proper sql type, needed to use named params
func vals2nameds(vals map[string]interface{}) []interface{} {
	var nameds []interface{}
	for key, val := range vals {
		nameds = append(nameds, sql.Named(key, val))
	}
	return nameds
}

func isEmptyRaw(raw json.RawMessage) bool {
	// the last check is for `null`
	return len(raw) == 0 || slices.Equal(raw, []byte{110, 117, 108, 108})
}

func raw2params(raw json.RawMessage) (*requestParams, error) {
	params := requestParams{}
	if isEmptyRaw(raw) {
		params.UnmarshalledArray = []any{}
		return &params, nil
	}
	switch raw[0] {
	case '[':
		values := make([]any, 0)
		err := json.Unmarshal(raw, &values)
		if err != nil {
			return nil, err
		}
		params.UnmarshalledArray = values
	case '{':
		values := make(map[string]interface{})
		err := json.Unmarshal(raw, &values)
		if err != nil {
			return nil, err
		}
		params.UnmarshalledDict = values
	default:
		return nil, errors.New("values should be an array or an object")
	}

	return &params, nil
}

// Processes paths with home (tilde) expansion. Fails if not valid
func expandHomeDir(path string, desc string) string {
	ePath, err := homedir.Expand(path)
	if err != nil {
		mllog.Fatalf("in expanding %s path: %s", desc, err.Error())
	}
	return ePath
}

// Crude but effective, I guess. At least, it's optimal for what I use it for: understand if a colon in second place is
// a drive separator or not
func isWindows() bool {
	abshere, err := filepath.Abs(".") // in docker, this is "/"
	if err != nil {
		mllog.Fatalf("Error in OS detection: %s", err)
	}
	return len(abshere) > 1 && bytes.Runes([]byte(abshere))[1] == ':'
}

var isWin = isWindows()

// Curiously, Go seems to lack an indexOf that allows to specify a starting point
func indexRuneAfter(haystack string, needle rune, after int) int {
	runes := bytes.Runes([]byte(haystack))
	for idx := after + 1; idx < len(runes); idx++ {
		if runes[idx] == needle {
			return idx
		}
	}
	return -1
}

// Returns the first two components of a column-delimited string; if there's no column, second is ""
// On windows, skips the first ':' if it's after one rune, then take everything after the first colon
func splitOnColon(toSplit string) (string, string) {
	after := -1
	if isWin {
		after = 1
	}
	if pos := indexRuneAfter(toSplit, ':', after); pos >= 0 {
		return toSplit[:pos], toSplit[pos+1:]
	}
	return toSplit, ""
}

func getDefault[T any](m orderedmap.OrderedMap, key string) T {
	value, ok := m.Get(key)
	if !ok {
		var t T
		return t
	}

	return value.(T)
}
