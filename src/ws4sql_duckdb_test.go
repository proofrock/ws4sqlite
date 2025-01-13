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
	"encoding/json"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	mllog "github.com/proofrock/go-mylittlelogger"

	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

func TestDDBSetupReg(t *testing.T) {
	os.Remove("../test/test.db")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type: utils.Ptr("DUCKDB"),
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
				},
				StoredStatement: []structs.StoredStatement{
					{
						Id:  "Q",
						Sql: "SELECT 1",
					},
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)

	if !utils.FileExists("../test/test.db") {
		t.Error("db file not created")
		return
	}
}

func TestDDBCreate(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}
}

func TestDDBFail(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
			},
		},
	}

	code, _, _ := call("test", req, t)

	if code != 500 {
		t.Error("did succeed, but shouldn't")
	}
}

func TestDDBNoFailIsIllegal(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'TWO')",
				NoFail:    true,
			},
		},
	}

	code, _, _ := call("test", req, t)

	if code != 400 {
		t.Error("should have given a response of 400")
		return
	}
}

func TestDDBTx(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
			{
				Query: "SELECT * FROM T1 WHERE ID = 1",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES ($ID, $AL)",
				Values: mkRaw(map[string]interface{}{
					"ID":  2,
					"VAL": "TWO",
				}),
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES ($ID, $VAL)",
				ValuesBatch: []json.RawMessage{
					mkRaw(map[string]interface{}{
						"ID":  3,
						"VAL": "THREE",
					}),
					mkRaw(map[string]interface{}{
						"ID":  4,
						"VAL": "FOUR",
					})},
			},
			{
				Query: "SELECT * FROM T1 WHERE ID > $ID",
				Values: mkRaw(map[string]interface{}{
					"ID": 0,
				}),
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success || *res.Results[0].RowsUpdated != 1 {
		t.Error("req 0 inconsistent")
	}

	if !res.Results[1].Success || utils.GetDefault[string](res.Results[1].ResultSet[0], "VAL") != "ONE" {
		t.Error("req 1 inconsistent")
	}

	if !res.Results[2].Success || *res.Results[2].RowsUpdated != 1 {
		t.Error("req 2 inconsistent")
	}

	if !res.Results[3].Success || len(res.Results[3].RowsUpdatedBatch) != 2 {
		t.Error("req 3 inconsistent")
	}

	if !res.Results[4].Success || len(res.Results[4].ResultSet) != 4 {
		t.Error("req 4 inconsistent")
	}
}

func TestDDBTxRollback(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "DELETE FROM T1",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 500 {
		t.Error("did succeed, but should have not")
		return
	}

	req = structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT * FROM T1",
			},
		},
	}

	code, _, res = call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success || len(res.Results[0].ResultSet) != 4 {
		t.Error("req 0 inconsistent")
	}
}

func TestDDBSQ(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "#Q",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}
}

func TestDDBConcurrent(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				// DuckDB enforces primary key constraints for the entire transaction, not
				// just at the point of execution of each statement, so this COMMIT is needed
				Statement: "DELETE FROM T1; COMMIT; INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
			{
				Query: "SELECT * FROM T1 WHERE ID = 1",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES ($ID, $VAL)",
				Values: mkRaw(map[string]interface{}{
					"ID":  2,
					"VAL": "TWO",
				}),
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES ($ID, $VAL)",
				ValuesBatch: []json.RawMessage{
					mkRaw(map[string]interface{}{
						"ID":  3,
						"VAL": "THREE",
					}),
					mkRaw(map[string]interface{}{
						"ID":  4,
						"VAL": "FOUR",
					})},
			},
			{
				Query: "SELECT * FROM T1 WHERE ID > $ID",
				Values: mkRaw(map[string]interface{}{
					"ID": 0,
				}),
			},
		},
	}

	wg := new(sync.WaitGroup)
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(t *testing.T) {
			defer wg.Done()
			code, body, res := call("test", req, t)

			if code != 200 {
				t.Errorf("did not succeed, code was %d - %s", code, body)
				return
			}

			if !res.Results[0].Success || *res.Results[0].RowsUpdated != 1 {
				t.Error("req 0 inconsistent")
			}

			if !res.Results[1].Success || utils.GetDefault[string](res.Results[1].ResultSet[0], "VAL") != "ONE" {
				t.Error("req 1 inconsistent")
			}

			if !res.Results[2].Success || *res.Results[2].RowsUpdated != 1 {
				t.Error("req 2 inconsistent")
			}

			if !res.Results[3].Success || len(res.Results[3].RowsUpdatedBatch) != 2 {
				t.Error("req 3 inconsistent")
			}

			if !res.Results[4].Success || len(res.Results[4].ResultSet) != 4 {
				t.Error("req 4 inconsistent")
			}
		}(t)
	}
	wg.Wait()
}

func TestDDBResultSetOrder(t *testing.T) {
	// See this issue for more context: https://github.com/proofrock/sqliterg/issues/5
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE table_with_many_columns (d INT, c INT, b INT, a INT)",
			},
			{
				Statement: "INSERT INTO table_with_many_columns VALUES (4, 3, 2, 1)",
			},
			{
				Query: "SELECT * FROM table_with_many_columns",
			},
			{
				Statement: "DROP TABLE table_with_many_columns",
			},
		},
	}
	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success ||
		!res.Results[1].Success ||
		!res.Results[2].Success ||
		!res.Results[3].Success {
		t.Error("did not succeed")
		return
	}

	queryResult := res.Results[2].ResultSet[0]
	expectedKeys := []string{"d", "c", "b", "a"}
	if !slices.Equal(
		queryResult.Keys(),
		expectedKeys,
	) {
		t.Error("should have the right order")
		return
	}

	expectedValues := []float64{4, 3, 2, 1}
	for i, key := range expectedKeys {
		value, ok := queryResult.Get(key)
		if !ok {
			t.Error("unreachable code")
			return
		}
		expectedValue := expectedValues[i]
		if value != expectedValue {
			t.Error("wrong value")
			return
		}
	}
}

func TestDDBListResultSet(t *testing.T) {
	// See this issue for more context: https://github.com/proofrock/sqliterg/issues/5
	req := structs.Request{
		ResultFormat: &listResults,
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE table_with_many_columns (d INT, c INT, b INT, a INT)",
			},
			{
				Statement: "INSERT INTO table_with_many_columns VALUES (4, 3, 2, 1)",
			},
			{
				Query: "SELECT * FROM table_with_many_columns",
			},
			{
				Statement: "DROP TABLE table_with_many_columns",
			},
		},
	}
	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success ||
		!res.Results[1].Success ||
		!res.Results[2].Success ||
		!res.Results[3].Success {
		t.Error("did not succeed")
		return
	}

	queryResult := res.Results[2].ResultSetList[0]
	queryResultHeaders := res.Results[2].ResultHeaders
	expectedKeys := []string{"d", "c", "b", "a"}
	if !slices.Equal(
		queryResultHeaders,
		expectedKeys,
	) {
		t.Error("should have the right order")
		return
	}

	expectedValues := []float64{4, 3, 2, 1}
	for i, key := range expectedKeys {
		value := queryResult[slices.Index(queryResultHeaders, key)]
		expectedValue := expectedValues[i]
		if value != expectedValue {
			t.Error("wrong value")
			return
		}
	}
}

func TestDDBArrayParams(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE table_with_many_columns (d INT, c INT, b INT, a INT)",
			},
			{
				Statement: "INSERT INTO table_with_many_columns VALUES (?, ?, ?, ?)",
				Values:    mkRaw([]int{1, 1, 1, 1}),
			},
			{
				Statement: "INSERT INTO table_with_many_columns VALUES (?, ?, ?, ?)",
				ValuesBatch: []json.RawMessage{
					mkRaw([]int{2, 2, 2, 2}),
					mkRaw([]int{3, 3, 3, 3}),
					mkRaw([]int{4, 4, 4, 4}),
				},
			},
			{
				Query: "SELECT * FROM table_with_many_columns",
			},
			{
				Statement: "DROP TABLE table_with_many_columns",
			},
		},
	}
	code, _, resp := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}
	queryResult := resp.Results[3]
	if !queryResult.Success {
		t.Error("could not query")
		return
	}
	records := queryResult.ResultSet
	if len(records) != 4 {
		t.Error("expected 4 records")
		return
	}
}

// don't remove the file, we'll use it for the next tests for read-only
func TestDDBTeardown(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

// Tests for read-only connections

func TestDDBSetupRO(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					Path:     utils.Ptr("../test/test.db"),
					ReadOnly: true,
				},
				StoredStatement: []structs.StoredStatement{
					{
						Id:  "Q",
						Sql: "SELECT 1",
					},
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestDDBFailRO(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
			},
		},
	}

	code, _, _ := call("test", req, t)

	if code != 500 {
		t.Error("did succeed, but shouldn't")
	}
}

func TestDDBOkRO(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT * FROM T1 ORDER BY ID ASC",
			},
		},
	}

	code, body, res := call("test", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}

	if !res.Results[0].Success || utils.GetDefault[string](res.Results[0].ResultSet[3], "VAL") != "FOUR" {
		t.Error("req is inconsistent")
	}
}

func TestDDBConcurrentRO(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT * FROM T1 ORDER BY ID ASC",
			},
		},
	}

	wg := new(sync.WaitGroup)
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(t *testing.T) {
			defer wg.Done()
			code, body, res := call("test", req, t)

			if code != 200 {
				t.Errorf("did not succeed, code was %d - %s", code, body)
				return
			}

			if !res.Results[0].Success || utils.GetDefault[string](res.Results[0].ResultSet[3], "VAL") != "FOUR" {
				t.Error("req is inconsistent")
			}
		}(t)
	}
	wg.Wait()
}

func TestDDBTeardownRO(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

// Tests for stored-statements-only connections

func TestDDBSetupSQO(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					Path:     utils.Ptr("../test/test.db"),
					ReadOnly: true,
				},
				UseOnlyStoredStatements: true,
				StoredStatement: []structs.StoredStatement{
					{
						Id:  "Q",
						Sql: "SELECT 1",
					},
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestDDBFailSQO(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "SELECT 1",
			},
		},
	}

	code, _, _ := call("test", req, t)

	if code != 400 {
		t.Error("did succeed, but shouldn't")
	}
}

func TestDDBOkSQO(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "#Q",
			},
		},
	}

	code, body, res := call("test", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}

	if !res.Results[0].Success {
		t.Error("req is inconsistent")
	}
}

func TestDDBTeardownSQO(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
	os.Remove("../test/test.db")
}

func TestDDBSetupMEM(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				StoredStatement: []structs.StoredStatement{
					{
						Id:  "Q",
						Sql: "SELECT 1",
					},
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestDDBMEM(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}
}

func TestDDBMEMIns(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}
}

func TestDDBTeardownMEM(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

func TestDDBSetupMEM_RO(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
					ReadOnly: false,
				},
				StoredStatement: []structs.StoredStatement{
					{
						Id:  "Q",
						Sql: "SELECT 1",
					},
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestDDBMEM_RO(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}
}

func TestDDBTeardownMEM_RO(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

// FIXME why fails?
// func TestRO_MEM_IS(t *testing.T) {
// 	// checks if it's possible to create a read only db with init statements (it shouldn't)
// 	cfg := structs.Config{
// 		Bindhost: "0.0.0.0",
// 		Port:     12321,
// 		Databases: []structs.Db{
// 			{
// 				DatabaseDef: structs.DatabaseDef{
// 					Type:     utils.Ptr("DUCKDB"),
// 					Id:       utils.Ptr("test"),
// 					InMemory: utils.Ptr(true),
// 					ReadOnly: true,
// 				},
// 				InitStatements: []string{
// 					"CREATE TABLE T1 (ID INT)",
// 				},
// 			},
// 		},
// 	}
// 	success := true
// 	mllog.WhenFatal = func(msg string) { success = false }
// 	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()
// 	go launch(cfg, true)
// 	time.Sleep(time.Second)
// 	Shutdown()
// 	if success {
// 		t.Error("did succeed, but shouldn't have")
// 	}
// }

func TestDDB_IS_Err(t *testing.T) {
	// checks if it exists after a failed init statement
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE T1 (ID INT)",
					"CREATE TABLE T1 (ID INT)",
				},
			},
		},
	}
	success := true
	mllog.WhenFatal = func(msg string) { success = false }
	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()
	go launch(cfg, true)
	time.Sleep(time.Second)
	Shutdown()
	if success {
		t.Error("did succeed, but shouldn't have")
	}
}

func TestDDB_DoubleId_Err(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
			}, {
				DatabaseDef: structs.DatabaseDef{
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
			},
		},
	}
	success := true
	mllog.WhenFatal = func(msg string) { success = false }
	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()
	go launch(cfg, true)
	time.Sleep(time.Second)
	Shutdown()
	if success {
		t.Error("did succeed, but shouldn't have")
	}
}

func TestDDB_DelWhenInitFails(t *testing.T) {
	defer Shutdown()
	defer os.Remove("../test/test.db")
	defer os.Remove("../test/test.wal")
	os.Remove("../test/test.db")
	os.Remove("../test/test.wal")

	mllog.WhenFatal = func(msg string) {}
	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type: utils.Ptr("DUCKDB"),
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
				},
				InitStatements: []string{
					"CLEARLY INVALID SQL",
				},
			},
		},
	}
	go launch(cfg, true)
	time.Sleep(time.Second)

	if utils.FileExists("../test/test.db") {
		t.Error("file wasn't cleared")
	}
}

// // If I put a question mark in the path, it must not interfere with the
// // ability to check if it's a new file. The second creation below
// // should NOT fail, as it's not a new file.
func TestDDB_CreateWithQuestionMark(t *testing.T) {
	defer Shutdown()
	defer os.Remove("../test/test.db")
	defer os.Remove("../test/test.wal")
	os.Remove("../test/test.db")
	os.Remove("../test/test.wal")

	success := true

	mllog.WhenFatal = func(msg string) { success = false }
	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type: utils.Ptr("DUCKDB"),
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
				},
				InitStatements: []string{
					"CREATE TABLE T1 (ID INT)",
				},
			},
		},
	}

	go launch(cfg, true)
	time.Sleep(time.Second)
	Shutdown()

	if !success {
		t.Error("did not succeed, but should have")
	}

	cfg = structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type: utils.Ptr("DUCKDB"),
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
				},
				InitStatements: []string{
					"CREATE TABLE T1 (ID INT)",
				},
			},
		},
	}
	go launch(cfg, true)
	time.Sleep(time.Second)

	if !success {
		t.Error("did not succeed, but should have")
	}
}

func TestDDBTwoServesOneDb(t *testing.T) {
	defer Shutdown()
	defer os.Remove("../test/test.db")
	defer os.Remove("../test/test.wal")
	os.Remove("../test/test.db")
	os.Remove("../test/test.wal")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type: utils.Ptr("DUCKDB"),
					Id:   utils.Ptr("test1"),
					Path: utils.Ptr("../test/test.db"),
				},
				InitStatements: []string{
					"CREATE TABLE T (NUM INT)",
				},
			}, {
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test2"),
					Path:     utils.Ptr("../test/test.db"),
					ReadOnly: true,
				},
			},
		},
	}

	go launch(cfg, true)

	time.Sleep(time.Second)

	req1 := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T VALUES (25)",
			},
		},
	}
	req2 := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT COUNT(1) FROM T",
			},
		},
	}

	wg := new(sync.WaitGroup)
	wg.Add(concurrency * 2)

	for i := 0; i < concurrency; i++ {
		go func(t *testing.T) {
			defer wg.Done()
			code, body, _ := call("test1", req1, t)
			if code != 200 {
				t.Error("INSERT failed", body)
			}
		}(t)
		go func(t *testing.T) {
			defer wg.Done()
			code, body, _ := call("test2", req2, t)
			if code != 200 {
				t.Error("SELECT failed", body)
			}
		}(t)
	}

	wg.Wait()

	time.Sleep(time.Second)
}

// // Test about the various field of a ResponseItem being null
// // when not actually involved

func TestDDBItemFieldsSetup(t *testing.T) {
	os.Remove("../test/test.db")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestDDBItemFieldsEmptySelect(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1 WHERE 0 = 1",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}

	resItem := res.Results[0]

	if resItem.ResultSet == nil {
		t.Error("select result is nil")
	}

	if resItem.Error != "" {
		t.Error("error is not empty")
	}

	if resItem.RowsUpdated != nil {
		t.Error("rowsUpdated is not nil")
	}

	if resItem.RowsUpdatedBatch != nil {
		t.Error("rowsUpdatedBatch is not nil")
	}
}

func TestDDBItemFieldsInsert(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 VALUES (1, 'a')",
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}

	resItem := res.Results[0]

	if resItem.ResultSet != nil {
		t.Error("select result is not nil")
	}

	if resItem.Error != "" {
		t.Error("error is not empty")
	}

	if resItem.RowsUpdated == nil {
		t.Error("rowsUpdated is nil")
	}

	if resItem.RowsUpdatedBatch != nil {
		t.Error("rowsUpdatedBatch is not nil")
	}
}

func TestDDBItemFieldsInsertBatch(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 VALUES ($ID, $VAL)",
				ValuesBatch: []json.RawMessage{
					mkRaw(map[string]interface{}{
						"ID":  3,
						"VAL": "THREE",
					}),
					mkRaw(map[string]interface{}{
						"ID":  4,
						"VAL": "FOUR",
					})},
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if !res.Results[0].Success {
		t.Error("did not succeed")
	}

	resItem := res.Results[0]

	if resItem.ResultSet != nil {
		t.Error("select result is not nil")
	}

	if resItem.Error != "" {
		t.Error("error is not empty")
	}

	if resItem.RowsUpdated != nil {
		t.Error("rowsUpdated is not nil")
	}

	if resItem.RowsUpdatedBatch == nil {
		t.Error("rowsUpdatedBatch is nil")
	}
}

func TestDDBItemFieldsTeardown(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

func TestDDBUnicode(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE T (TXT TEXT)",
				},
			},
		},
	}

	go launch(cfg, true)

	time.Sleep(time.Second)

	req1 := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T VALUES ('世界')",
			},
		},
	}
	req2 := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT TXT FROM T",
			},
		},
	}

	code, body, _ := call("test", req1, t)
	if code != 200 {
		t.Error("INSERT failed", body)
	}

	code, body, res := call("test", req2, t)
	if code != 200 {
		t.Error("SELECT failed", body)
	}
	if utils.GetDefault[string](res.Results[0].ResultSet[0], "TXT") != "世界" {
		t.Error("Unicode extraction failed", body)
	}

	time.Sleep(time.Second)

	Shutdown()
}

func TestDDBFailBegin(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type:     utils.Ptr("DUCKDB"),
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE T (TXT TEXT)",
				},
			},
		},
	}

	go launch(cfg, true)

	time.Sleep(time.Second)

	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "BEGIN",
			},
		},
	}

	code, _, _ := call("test", req, t)

	if code == 200 {
		t.Error("structs.Request succeeded, but shouldn't have")
	}

	req = structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "COMMIT",
			},
		},
	}

	code, _, _ = call("test", req, t)

	if code == 200 {
		t.Error("structs.Request succeeded, but shouldn't have")
	}

	req = structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "ROLLBACK",
			},
		},
	}

	code, _, _ = call("test", req, t)

	if code == 200 {
		t.Error("structs.Request succeeded, but shouldn't have")
	}

	time.Sleep(time.Second)

	Shutdown()
}

func TestDDBExoticSuffixes(t *testing.T) {
	os.Remove("../test/test.duckdb")
	defer os.Remove("../test/test.duckdb")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Type: utils.Ptr("DUCKDB"),
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.duckdb"),
				},
				StoredStatement: []structs.StoredStatement{
					{
						Id:  "Q",
						Sql: "SELECT 1",
					},
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)

	if !utils.FileExists("../test/test.duckdb") {
		t.Error("db file not created")
		return
	}

	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
			},
		},
	}

	code, _, _ := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
	}

	time.Sleep(time.Second)

	Shutdown()
}
