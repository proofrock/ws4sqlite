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

package utils

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"

	"github.com/iancoleman/orderedmap"
	"github.com/mitchellh/go-homedir"
	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
)

// Uppercases (?) the first letter of a string
func Capitalize(str string) string {
	return strings.ToUpper(str[0:1]) + str[1:]
}

// Does a file exist? No error returned.
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		mllog.Fatal("in stating file '", filename, "': ", err.Error())
	}
	return !info.IsDir()
}

// Does a dir exist? No error returned.
func DirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		mllog.Fatal("in stating dir '", dirname, "': ", err.Error())
	}
	return info.IsDir()
}

// Maps named parameters to the proper sql type, needed to use named params
func Vals2nameds(vals map[string]interface{}) []interface{} {
	var nameds []interface{}
	for key, val := range vals {
		nameds = append(nameds, sql.Named(key, val))
	}
	return nameds
}

func IsEmptyRaw(raw json.RawMessage) bool {
	// the last check is for `null`
	return len(raw) == 0 || slices.Equal(raw, []byte{110, 117, 108, 108})
}

func Raw2params(raw json.RawMessage) (*structs.RequestParams, error) {
	params := structs.RequestParams{}
	if IsEmptyRaw(raw) {
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
func ExpandHomeDir(path string, desc string) string {
	ePath, err := homedir.Expand(path)
	if err != nil {
		mllog.Fatalf("in expanding %s path: %s", desc, err.Error())
	}
	return ePath
}

func Ptr[T any](str T) *T {
	val := str
	return &val
}

func GetDefault[T any](m orderedmap.OrderedMap, key string) T {
	value, ok := m.Get(key)
	if !ok {
		var t T
		return t
	}

	return value.(T)
}
