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

package structs

import (
	"encoding/json"

	"github.com/iancoleman/orderedmap"
)

// These are for parsing the request (from JSON)

type Credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type RequestItem struct {
	Query       string            `json:"query"`
	Statement   string            `json:"statement"`
	NoFail      bool              `json:"noFail"`
	Values      json.RawMessage   `json:"values"`
	ValuesBatch []json.RawMessage `json:"valuesBatch"`
}

type Request struct {
	ResultFormat *string       `json:"resultFormat"`
	Credentials  *Credentials  `json:"credentials"`
	Transaction  []RequestItem `json:"transaction"`
}

type RequestParams struct {
	UnmarshalledDict  map[string]any
	UnmarshalledArray []any
}

// These are for generating the response
type ResponseItem struct {
	Success          bool                    `json:"success"`
	RowsUpdated      *int64                  `json:"rowsUpdated,omitempty"`
	RowsUpdatedBatch []int64                 `json:"rowsUpdatedBatch,omitempty"`
	ResultHeaders    []string                `json:"resultHeaders,omitempty"`
	ResultSet        []orderedmap.OrderedMap `json:"resultSet,omitnil"`     // omitnil is used by jettison
	ResultSetList    [][]interface{}         `json:"resultSetList,omitnil"` // omitnil is used by jettison
	Error            string                  `json:"error,omitempty"`
}

type Response struct {
	Results []ResponseItem `json:"results"`
}
