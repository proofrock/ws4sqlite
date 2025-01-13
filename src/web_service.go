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
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/iancoleman/orderedmap"
	"github.com/proofrock/ws4sql/engines"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"

	"github.com/gofiber/fiber/v2"
)

// Common interface for db.Conn and db.Tx
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

var ctx = context.Background()

// Catches the panics and converts the argument in a struct that Fiber uses to
// signal the error, setting the structs.Response code and the JSON that is actually returned
// with all its properties.
//
// It uses <panic> and the recover middleware to manage errors because it's the only
// way I know to let a custom structure/error arrive here; the standard way can only
// wrap a string.
func errHandler(c *fiber.Ctx, err error) error {
	var ret structs.WsError

	// Converts all the possible errors that arrive here to a structs.WSError
	if fe, ok := err.(*fiber.Error); ok {
		ret = structs.NewWSError(-1, fe.Code, "%s", utils.Capitalize(fe.Error()))
	} else if wse, ok := err.(structs.WsError); ok {
		ret = wse
	} else {
		ret = structs.NewWSError(-1, fiber.StatusInternalServerError, "%s", utils.Capitalize(err.Error()))
	}

	return c.Status(ret.Code).JSON(ret)
}

// For a single query item, deals with a failure, determining if it must invalidate all of the transaction
// or just report an error in the single query. In the former case, fails fast (panics), else it appends
// the error to the structs.Response items, so the caller needs to return7continue
func reportError(err error, code int, reqIdx int, noFail bool, results []structs.ResponseItem) {
	if !noFail {
		panic(structs.NewWSError(reqIdx, code, "%s", err.Error()))
	}
	results[reqIdx] = structs.ResponseItem{
		Success:          false,
		RowsUpdated:      nil,
		RowsUpdatedBatch: nil,
		ResultHeaders:    nil,
		ResultSet:        nil,
		ResultSetList:    nil,
		Error:            utils.Capitalize(err.Error()),
	}
}

// Processes a query, and returns a suitable structs.ResponseItem
//
// This method is needed to execute properly the defers.
func processWithResultSet(tx *DBExecutor, query string, isListResultSet bool, params structs.RequestParams) (*structs.ResponseItem, error) {
	resultSet := make([]orderedmap.OrderedMap, 0)
	resultSetList := make([][]interface{}, 0)

	rows := (*sql.Rows)(nil)
	err := (error)(nil)
	if params.UnmarshalledDict == nil && params.UnmarshalledArray == nil {
		rows, err = nil, errors.New("processWithResultSet unreachable code")
	} else if params.UnmarshalledDict != nil {
		rows, err = (*tx).QueryContext(ctx, query, utils.Vals2nameds(params.UnmarshalledDict)...)
	} else {
		rows, err = (*tx).QueryContext(ctx, query, params.UnmarshalledArray...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	headers, _ := rows.Columns() // I can ignore the error, rows aren't closed
	for rows.Next() {
		values := make([]interface{}, len(headers)) // values of the various fields
		scans := make([]interface{}, len(headers))  // pointers to the values, to pass to Scan()
		for i := range values {
			scans[i] = &values[i]
		}
		if err = rows.Scan(scans...); err != nil {
			return nil, err
		}

		if isListResultSet {
			// List-style result set

			resultSetList = append(resultSetList, values)
		} else {
			// Map-style result set

			toAdd := orderedmap.New()
			for i := range values {
				toAdd.Set(headers[i], values[i])
			}

			resultSet = append(resultSet, *toAdd)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if isListResultSet {
		return &structs.ResponseItem{
			Success:          true,
			RowsUpdated:      nil,
			RowsUpdatedBatch: nil,
			ResultHeaders:    headers,
			ResultSet:        nil,
			ResultSetList:    resultSetList,
			Error:            "",
		}, nil
	}
	return &structs.ResponseItem{
		Success:          true,
		RowsUpdated:      nil,
		RowsUpdatedBatch: nil,
		ResultHeaders:    headers,
		ResultSet:        resultSet,
		ResultSetList:    nil,
		Error:            "",
	}, nil
}

// Process a single statement, and returns a suitable structs.ResponseItem
func processForExec(tx *DBExecutor, statement string, params structs.RequestParams) (*structs.ResponseItem, error) {
	res := (sql.Result)(nil)
	err := (error)(nil)
	if params.UnmarshalledDict == nil && params.UnmarshalledArray == nil {
		res, err = nil, errors.New("processWithResultSet unreachable code")
	} else if params.UnmarshalledDict != nil {
		res, err = (*tx).ExecContext(ctx, statement, utils.Vals2nameds(params.UnmarshalledDict)...)
	} else {
		res, err = (*tx).ExecContext(ctx, statement, params.UnmarshalledArray...)
	}
	if err != nil {
		return nil, err
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	return &structs.ResponseItem{
		Success:          true,
		RowsUpdated:      &rowsUpdated,
		RowsUpdatedBatch: nil,
		ResultHeaders:    nil,
		ResultSet:        nil,
		ResultSetList:    nil,
		Error:            "",
	}, nil
}

// Process a batch statement, and returns a suitable structs.ResponseItem.
// It prepares the statement, then executes it for each of the values' sets.
func processForExecBatch(tx *DBExecutor, q string, paramsBatch []structs.RequestParams) (*structs.ResponseItem, error) {
	ps, err := (*tx).PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer ps.Close()

	var rowsUpdatedBatch []int64
	for _, params := range paramsBatch {
		res := (sql.Result)(nil)
		err := (error)(nil)
		if params.UnmarshalledDict == nil && params.UnmarshalledArray == nil {
			res, err = nil, errors.New("processWithResultSet unreachable code")
		} else if params.UnmarshalledDict != nil {
			res, err = (*tx).ExecContext(ctx, q, utils.Vals2nameds(params.UnmarshalledDict)...)
		} else {
			res, err = (*tx).ExecContext(ctx, q, params.UnmarshalledArray...)
		}
		if err != nil {
			return nil, err
		}

		rowsUpdated, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}

		rowsUpdatedBatch = append(rowsUpdatedBatch, rowsUpdated)
	}

	return &structs.ResponseItem{
		Success:          true,
		RowsUpdated:      nil,
		RowsUpdatedBatch: rowsUpdatedBatch,
		ResultHeaders:    nil,
		ResultSet:        nil,
		ResultSetList:    nil,
		Error:            "",
	}, nil
}

func ckSQL(sql string) string {
	if strings.HasPrefix(strings.ToUpper(sql), "BEGIN") {
		return "BEGIN is not allowed"
	}
	if strings.HasPrefix(strings.ToUpper(sql), "COMMIT") {
		return "COMMIT is not allowed"
	}
	if strings.HasPrefix(strings.ToUpper(sql), "ROLLBACK") {
		return "ROLLBACK is not allowed"
	}
	return ""
}

// Handler for the POST. Receives the body of the HTTP request, parses it
// and executes the transaction on the database retrieved from the URL path.
// Constructs and sends the structs.Response.
func handler(databaseId string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var body structs.Request
		if err := c.BodyParser(&body); err != nil {
			return structs.NewWSError(-1, fiber.StatusBadRequest, "in parsing body: %s", err.Error())
		}

		isListResultSet := body.ResultFormat != nil && strings.EqualFold(*body.ResultFormat, "list")

		db, found := dbs[databaseId]
		if !found {
			return structs.NewWSError(-1, fiber.StatusNotFound, "database with ID '%s' not found", databaseId)
		}

		// Fail fast if empty
		if len(body.Transaction) == 0 {
			return structs.NewWSError(-1, fiber.StatusBadRequest, "missing statements list ('transaction' node)")
		}

		// Static validation of the request, fails fast. This is done by database type: in general, we
		// are looking for instructions that aren't supported by a certain database type.
		// FIXME refactor
		staticCheckErr := engines.GetFlavorForDb(db).CheckRequest(body)
		if staticCheckErr != nil {
			return *staticCheckErr
		}

		// Execute non-concurrently
		db.Mutex.Lock()
		defer db.Mutex.Unlock()

		if db.Auth != nil && strings.ToUpper(db.Auth.Mode) == authModeInline {
			if err := applyAuth(&db, &body); err != nil {
				// When unauthenticated waits for 1s to hinder brute force attacks
				time.Sleep(time.Second)
				if db.Auth.CustomErrorCode != nil {
					return structs.NewWSError(-1, *db.Auth.CustomErrorCode, "%s", err.Error())
				}
				return structs.NewWSError(-1, fiber.StatusUnauthorized, "%s", err.Error())
			}
		}

		// Opens a transaction. One more occasion to specify: read only ;-)
		var dbExecutor DBExecutor
		useTransaction := *db.DatabaseDef.Type != engines.ID_DUCKDB || !db.DatabaseDef.ReadOnly
		if useTransaction {
			var err error
			dbExecutor, err = db.DbConn.BeginTx(
				context.Background(),
				&sql.TxOptions{
					Isolation: engines.GetFlavorForDb(db).GetDefaultIsolationLevel(),
					ReadOnly:  db.DatabaseDef.ReadOnly,
				},
			)
			if err != nil {
				return structs.NewWSError(-1, fiber.StatusInternalServerError, "%s", err.Error())
			}
		} else {
			dbExecutor = db.DbConn
		}

		tainted := true // If I reach the end of the method, I switch this to false to signal success
		defer func() {
			if useTransaction {
				var tx = dbExecutor.(*sql.Tx)
				if tainted {
					tx.Rollback()
				} else {
					tx.Commit()
				}
			}
		}()

		var ret structs.Response
		ret.Results = make([]structs.ResponseItem, len(body.Transaction))

		for i := range body.Transaction {
			txItem := body.Transaction[i]

			if (txItem.Query == "") == (txItem.Statement == "") {
				reportError(errors.New("one and only one of query or statement must be provided"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
				continue
			}

			hasResultSet := txItem.Query != ""

			if !utils.IsEmptyRaw(txItem.Values) && len(txItem.ValuesBatch) != 0 {
				reportError(errors.New("cannot specify both values and valuesBatch"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
				continue
			}

			if hasResultSet && len(txItem.ValuesBatch) > 0 {
				reportError(errors.New("cannot specify valuesBatch for queries (only for statements)"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
				continue
			}

			var sqll string

			if hasResultSet {
				sqll = txItem.Query
			} else {
				sqll = txItem.Statement
			}

			// Sanitize: BEGIN, COMMIT and ROLLBACK aren't allowed
			if errStr := ckSQL(sqll); errStr != "" {
				reportError(errors.New("errStr"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
				continue
			}

			// Processes a stored statement
			if strings.HasPrefix(sqll, "#") {
				var ok bool
				sqll, ok = db.StoredStatsMap[sqll[1:]]
				if !ok {
					reportError(errors.New("a stored statement is required, but did not find it"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
					continue
				}
			} else {
				if db.UseOnlyStoredStatements {
					reportError(errors.New("configured to serve only stored statements, but SQL is passed"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
					continue
				}
			}

			if len(txItem.ValuesBatch) > 0 {
				// Process a batch statement (multiple values)
				var paramsBatch []structs.RequestParams
				for i2 := range txItem.ValuesBatch {
					params, err := utils.Raw2params(txItem.ValuesBatch[i2])
					if err != nil {
						reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
						continue
					}

					paramsBatch = append(paramsBatch, *params)
				}

				retE, err := processForExecBatch(&dbExecutor, sqll, paramsBatch)
				if err != nil {
					reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
					continue
				}

				ret.Results[i] = *retE
			} else {
				// At most one values set (be it query or statement)
				params, err := utils.Raw2params(txItem.Values)
				if err != nil {
					reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
					continue
				}

				if hasResultSet {
					// Query
					// Externalized in a func so that defer rows.Close() actually runs
					retWR, err := processWithResultSet(&dbExecutor, sqll, isListResultSet, *params)
					if err != nil {
						reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
						continue
					}

					ret.Results[i] = *retWR
				} else {
					// Statement
					retE, err := processForExec(&dbExecutor, sqll, *params)
					if err != nil {
						reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
						continue
					}

					ret.Results[i] = *retE
				}
			}
		}

		tainted = false

		return c.Status(200).JSON(ret)
	}
}
