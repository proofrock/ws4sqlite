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

	"github.com/gofiber/fiber/v2"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

const concurrency = 64

func TestMain(m *testing.M) {
	println("Go...")
	oldLevel := mllog.Level
	mllog.Level = mllog.NOT_EVEN_STDERR
	exitCode := m.Run()
	mllog.Level = oldLevel
	println("...finished")
	os.Exit(exitCode)
}

func Shutdown() {
	stopScheduler()
	if len(dbs) > 0 {
		mllog.StdOut("Closing databases...")
		for i := range dbs {
			if dbs[i].DbConn != nil {
				dbs[i].DbConn.Close()
			}
			dbs[i].Db.Close()
			delete(dbs, i)
		}
	}
	if app != nil {
		mllog.StdOut("Shutting down web server...")
		app.Shutdown()
		app = nil
	}
}

// call with basic auth support
func callBA(databaseId string, req structs.Request, user, password string, t *testing.T) (int, string, structs.Response) {
	json_data, err := json.Marshal(req)
	if err != nil {
		t.Error(err)
	}

	client := &fiber.Client{}
	post := client.Post("http://localhost:12321/"+databaseId).
		Body(json_data).
		Set("Content-Type", "application/json")

	if user != "" {
		post = post.BasicAuth(user, password)
	}

	code, bodyBytes, errs := post.Bytes()

	if len(errs) > 0 {
		t.Error(errs[0])
	}

	var res structs.Response
	if err := json.Unmarshal(bodyBytes, &res); code == 200 && err != nil {
		println(string(bodyBytes))
		t.Error(err)
	}
	return code, string(bodyBytes), res
}

func call(databaseId string, req structs.Request, t *testing.T) (int, string, structs.Response) {
	return callBA(databaseId, req, "", "", t)
}

func mkRaw(mapp any) json.RawMessage {
	bs, _ := json.Marshal(mapp)
	return bs
}

func TestSetupReg(t *testing.T) {
	os.Remove("../test/test.db")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
				},
				// DisableWALMode: true,
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

func TestCreate(t *testing.T) {
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

func TestFail(t *testing.T) {
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

func TestTx(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'TWO')",
				NoFail:    true,
			},
			{
				Query: "SELECT * FROM T1 WHERE ID = 1",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (:ID, :VAL)",
				Values: mkRaw(map[string]interface{}{
					"ID":  2,
					"VAL": "TWO",
				}),
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (:ID, :VAL)",
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
				Query: "SELECT * FROM T1 WHERE ID > :ID",
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

	if res.Results[1].Success {
		t.Error("req 1 inconsistent")
	}

	if !res.Results[2].Success || utils.GetDefault[string](res.Results[2].ResultSet[0], "VAL") != "ONE" {
		t.Error("req 2 inconsistent")
	}

	if !res.Results[3].Success || *res.Results[3].RowsUpdated != 1 {
		t.Error("req 3 inconsistent")
	}

	if !res.Results[4].Success || len(res.Results[4].RowsUpdatedBatch) != 2 {
		t.Error("req 4 inconsistent")
	}

	if !res.Results[5].Success || len(res.Results[5].ResultSet) != 4 {
		t.Error("req 5 inconsistent")
	}

}

func TestTxRollback(t *testing.T) {
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

func TestSQ(t *testing.T) {
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

func TestConcurrent(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "DELETE FROM T1; INSERT INTO T1 (ID, VAL) VALUES (1, 'ONE')",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (1, 'TWO')",
				NoFail:    true,
			},
			{
				Query: "SELECT * FROM T1 WHERE ID = 1",
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (:ID, :VAL)",
				Values: mkRaw(map[string]interface{}{
					"ID":  2,
					"VAL": "TWO",
				}),
			},
			{
				Statement: "INSERT INTO T1 (ID, VAL) VALUES (:ID, :VAL)",
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
				Query: "SELECT * FROM T1 WHERE ID > :ID",
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

			if res.Results[1].Success {
				t.Error("req 1 inconsistent")
			}

			if !res.Results[2].Success || utils.GetDefault[string](res.Results[2].ResultSet[0], "VAL") != "ONE" {
				t.Error("req 2 inconsistent")
			}

			if !res.Results[3].Success || *res.Results[3].RowsUpdated != 1 {
				t.Error("req 3 inconsistent")
			}

			if !res.Results[4].Success || len(res.Results[4].RowsUpdatedBatch) != 2 {
				t.Error("req 4 inconsistent")
			}

			if !res.Results[5].Success || len(res.Results[5].ResultSet) != 4 {
				t.Error("req 5 inconsistent")
			}
		}(t)
	}
	wg.Wait()
}

func TestResultSetOrder(t *testing.T) {
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

var listResults string = "list"

func TestListResultSet(t *testing.T) {
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

func TestArrayParams(t *testing.T) {
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
func TestTeardown(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

// Tests for read-only connections

func TestSetupRO(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
					// DisableWALMode: true,
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

func TestFailRO(t *testing.T) {
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

func TestOkRO(t *testing.T) {
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

func TestConcurrentRO(t *testing.T) {
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

func TestTeardownRO(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

// Tests for stored-statements-only connections

func TestSetupSQO(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.db"),
					// DisableWALMode: true,
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

func TestFailSQO(t *testing.T) {
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

func TestOkSQO(t *testing.T) {
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

func TestTeardownSQO(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
	os.Remove("../test/test.db")
}

func TestSetupMEM(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				// DisableWALMode: true,
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

func TestMEM(t *testing.T) {
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

func TestMEMIns(t *testing.T) {
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

func TestTeardownMEM(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

func TestSetupMEM_RO(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
					ReadOnly: true,
					// DisableWALMode: true,
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

func TestMEM_RO(t *testing.T) {
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

func TestTeardownMEM_RO(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

func TestSetupWITH_ADD_PROPS(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				// DisableWALMode: true,
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

func TestWITH_ADD_PROPS(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "CREATE TABLE T1 (ID INT PRIMARY KEY, VAL TEXT NOT NULL)",
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

func TestTeardownWITH_ADD_PROPS(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

func TestRO_MEM_IS(t *testing.T) {
	// checks if it's possible to create a read only db with init statements (it shouldn't)
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
					ReadOnly: true,
					// DisableWALMode: true,
				},
				InitStatements: []string{
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

func Test_IS_Err(t *testing.T) {
	// checks if it exists after a failed init statement
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:       utils.Ptr("test"),
					InMemory: utils.Ptr(true),
				},
				// DisableWALMode: true,
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

func Test_DoubleId_Err(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
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

func Test_DelWhenInitFails(t *testing.T) {
	defer Shutdown()
	defer os.Remove("../test/test.db")
	defer os.Remove("../test/test.db-shm")
	defer os.Remove("../test/test.db-wal")
	os.Remove("../test/test.db")
	os.Remove("../test/test.db-shm")
	os.Remove("../test/test.db-wal")

	mllog.WhenFatal = func(msg string) {}
	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
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

// If I put a question mark in the path, it must not interfere with the
// ability to check if it's a new file. The second creation below
// should NOT fail, as it's not a new file.
func Test_CreateWithQuestionMark(t *testing.T) {
	defer Shutdown()
	defer os.Remove("../test/test.db")
	defer os.Remove("../test/test.db-shm")
	defer os.Remove("../test/test.db-wal")
	os.Remove("../test/test.db")
	os.Remove("../test/test.db-shm")
	os.Remove("../test/test.db-wal")

	success := true

	mllog.WhenFatal = func(msg string) { success = false }
	defer func() { mllog.WhenFatal = func(msg string) { os.Exit(1) } }()

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
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

func TestTwoServesOneDb(t *testing.T) {
	defer Shutdown()
	defer os.Remove("../test/test.db")
	defer os.Remove("../test/test.db-shm")
	defer os.Remove("../test/test.db-wal")
	os.Remove("../test/test.db")
	os.Remove("../test/test.db-shm")
	os.Remove("../test/test.db-wal")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:   utils.Ptr("test1"),
					Path: utils.Ptr("../test/test.db"),
				},
				InitStatements: []string{
					"CREATE TABLE T (NUM INT)",
				},
			}, {
				DatabaseDef: structs.DatabaseDef{
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

// Test about the various field of a ResponseItem being null
// when not actually involved

func TestItemFieldsSetup(t *testing.T) {
	os.Remove("../test/test.db")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
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

func TestItemFieldsEmptySelect(t *testing.T) {
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

func TestItemFieldsInsert(t *testing.T) {
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

func TestItemFieldsInsertBatch(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Statement: "INSERT INTO T1 VALUES (:ID, :VAL)",
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

func TestItemFieldsError(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query:  "A CLEARLY INVALID SQL",
				NoFail: true,
			},
		},
	}

	code, _, res := call("test", req, t)

	if code != 200 {
		t.Error("did not succeed")
		return
	}

	if res.Results[0].Success {
		t.Error("did succeed, but it shoudln't have")
	}

	resItem := res.Results[0]

	if resItem.ResultSet != nil {
		t.Error("select result is not nil")
	}

	if resItem.Error == "" {
		t.Error("error is empty")
	}

	if resItem.RowsUpdated != nil {
		t.Error("rowsUpdated is not nil")
	}

	if resItem.RowsUpdatedBatch != nil {
		t.Error("rowsUpdatedBatch is not nil")
	}
}

func TestItemFieldsTeardown(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
}

func TestUnicode(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
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

func TestFailBegin(t *testing.T) {
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
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
				NoFail:    true,
				Statement: "BEGIN",
			},
			{
				NoFail:    true,
				Statement: "COMMIT",
			},
			{
				NoFail:    true,
				Statement: "ROLLBACK",
			},
		},
	}

	code, _, res := call("test", req, t)
	if code != 200 {
		t.Error("structs.Request failed, but shouldn't have")
	}

	if res.Results[0].Success {
		t.Error("BEGIN succeeds, but shouldn't have")
	}
	if res.Results[1].Success {
		t.Error("COMMIT succeeds, but shouldn't have")
	}
	if res.Results[2].Success {
		t.Error("ROLLBACK succeeds, but shouldn't have")
	}

	time.Sleep(time.Second)

	Shutdown()
}

func TestExoticSuffixes(t *testing.T) {
	os.Remove("../test/test.sqlite3")
	defer os.Remove("../test/test.sqlite3")

	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:   utils.Ptr("test"),
					Path: utils.Ptr("../test/test.sqlite3"),
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

	if !utils.FileExists("../test/test.sqlite3") {
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
