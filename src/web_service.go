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

	"github.com/gofiber/fiber/v2"
	"github.com/proofrock/crypgo"
)

// Catches the panics and converts the argument in a struct that Fiber uses to
// signal the error, setting the response code and the JSON that is actually returned
// with all its properties.
//
// It uses <panic> and the recover middleware to manage errors because it's the only
// way I know to let a custom structure/error arrive here; the standard way can only
// wrap a string.
func errHandler(c *fiber.Ctx, err error) error {
	var ret wsError

	// Converts all the possible errors that arrive here to a wsError
	if fe, ok := err.(*fiber.Error); ok {
		ret = newWSError(-1, fe.Code, capitalize(fe.Error()))
	} else if wse, ok := err.(wsError); ok {
		ret = wse
	} else {
		ret = newWSError(-1, fiber.StatusInternalServerError, capitalize(err.Error()))
	}

	return c.Status(ret.Code).JSON(ret)
}

// Scans the values for a db request and encrypts them as needed
func encrypt(encoder requestItemCrypto, values map[string]interface{}) error {
	for i := range encoder.Fields {
		sval, ok := values[encoder.Fields[i]].(string)
		if !ok {
			return errors.New("attempting to encrypt a non-string field")
		}
		var eval string
		var err error
		if encoder.CompressionLevel < 1 {
			eval, err = crypgo.Encrypt(encoder.Password, sval)
		} else if encoder.CompressionLevel < 20 {
			eval, err = crypgo.CompressAndEncrypt(encoder.Password, sval, encoder.CompressionLevel)
		} else {
			return errors.New("compression level is in the range 0-19")
		}
		if err != nil {
			return err
		}
		values[encoder.Fields[i]] = eval
	}
	return nil
}

// Scans the results from a db request and decrypts them as needed
func decrypt(decoder requestItemCrypto, results map[string]interface{}) error {
	if decoder.CompressionLevel > 0 {
		return errors.New("cannot specify compression level for decryption")
	}
	for i := range decoder.Fields {
		sval, ok := results[decoder.Fields[i]].(string)
		if !ok {
			return errors.New("attempting to decrypt a non-string field")
		}
		dval, err := crypgo.Decrypt(decoder.Password, sval)
		if err != nil {
			return err
		}
		results[decoder.Fields[i]] = dval
	}
	return nil
}

// For a single query item, deals with a failure, determining if it must invalidate all of the transaction
// or just report an error in the single query. In the former case, fails fast (panics), else it appends
// the error to the response items, so the caller needs to return7continue
func reportError(err error, code int, reqIdx int, noFail bool, results []responseItem) {
	if !noFail {
		panic(newWSError(reqIdx, code, err.Error()))
	}
	results[reqIdx] = responseItem{false, nil, nil, nil, capitalize(err.Error())}
}

// Processes a query, and returns a suitable responseItem
//
// This method is needed to execute properly the defers.
func processWithResultSet(tx *sql.Tx, query string, decoder *requestItemCrypto, values map[string]interface{}) (*responseItem, error) {
	resultSet := make([]map[string]interface{}, 0)

	rows, err := tx.Query(query, vals2nameds(values)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields, _ := rows.Columns() // I can ignore the error, rows aren't closed
	for rows.Next() {
		values := make([]interface{}, len(fields)) // values of the various fields
		scans := make([]interface{}, len(fields))  // pointers to the values, to pass to Scan()
		for i := range values {
			scans[i] = &values[i]
		}
		if err = rows.Scan(scans...); err != nil {
			return nil, err
		}

		toAdd := make(map[string]interface{})
		for i := range values {
			toAdd[fields[i]] = values[i]
		}

		if decoder != nil {
			if err := decrypt(*decoder, toAdd); err != nil {
				return nil, err
			}
		}
		resultSet = append(resultSet, toAdd)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &responseItem{true, nil, nil, resultSet, ""}, nil
}

// Process a single statement, and returns a suitable responseItem
func processForExec(tx *sql.Tx, statement string, values map[string]interface{}) (*responseItem, error) {
	res, err := tx.Exec(statement, vals2nameds(values)...)
	if err != nil {
		return nil, err
	}

	rowsUpdated, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	return &responseItem{true, &rowsUpdated, nil, nil, ""}, nil
}

// Process a batch statement, and returns a suitable responseItem.
// It prepares the statement, then executes it for each of the values' sets.
func processForExecBatch(tx *sql.Tx, q string, valuesBatch []map[string]interface{}) (*responseItem, error) {
	ps, err := tx.Prepare(q)
	if err != nil {
		return nil, err
	}
	defer ps.Close()

	var rowsUpdatedBatch []int64
	for i := range valuesBatch {
		res, err := ps.Exec(vals2nameds(valuesBatch[i])...)
		if err != nil {
			return nil, err
		}

		rowsUpdated, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}

		rowsUpdatedBatch = append(rowsUpdatedBatch, rowsUpdated)
	}

	return &responseItem{true, nil, rowsUpdatedBatch, nil, ""}, nil
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
// Constructs and sends the response.
func handler(c *fiber.Ctx) error {
	var body request
	if err := c.BodyParser(&body); err != nil {
		return newWSError(-1, fiber.StatusBadRequest, "in parsing body: %s", err.Error())
	}

	databaseId := c.Params("databaseId")
	if databaseId == "" {
		return newWSError(-1, fiber.StatusNotFound, "missing database ID")
	}

	db, found := dbs[databaseId]
	if !found {
		return newWSError(-1, fiber.StatusNotFound, "database with ID '%s' not found", databaseId)
	}

	// Execute non-concurrently
	db.Mutex.Lock()
	defer db.Mutex.Unlock()

	if db.Auth != nil && strings.ToUpper(db.Auth.Mode) == authModeInline {
		if err := applyAuth(&db, &body); err != nil {
			// When unauthenticated waits for 1s to hinder brute force attacks
			time.Sleep(time.Second)
			return newWSError(-1, fiber.StatusUnauthorized, err.Error())
		}
	}

	if len(body.Transaction) == 0 {
		return newWSError(-1, fiber.StatusBadRequest, "missing statements list ('transaction' node)")
	}

	dbConn, err := db.Db.Conn(context.Background())
	if err != nil {
		return newWSError(-1, fiber.StatusInternalServerError, err.Error())
	}
	defer dbConn.Close()

	// Opens a transaction. One more occasion to specify: read only ;-)
	tx, err := dbConn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: db.ReadOnly})
	if err != nil {
		return newWSError(-1, fiber.StatusInternalServerError, err.Error())
	}

	tainted := true // If I reach the end of the method, I switch this to false to signal success
	defer func() {
		if tainted {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var ret response
	ret.Results = make([]responseItem, len(body.Transaction))

	for i := range body.Transaction {
		txItem := body.Transaction[i]

		if (txItem.Query == "") == (txItem.Statement == "") {
			reportError(errors.New("one and only one of query or statement must be provided"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
			continue
		}

		hasResultSet := txItem.Query != ""

		if hasResultSet && txItem.Encoder != nil {
			reportError(errors.New("cannot specify an encoder for a query"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
			continue
		}

		if !hasResultSet && txItem.Decoder != nil {
			reportError(errors.New("cannot specify a decoder for a statement"), fiber.StatusBadRequest, i, txItem.NoFail, ret.Results)
			continue
		}

		if len(txItem.Values) != 0 && len(txItem.ValuesBatch) != 0 {
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
			var valuesBatch []map[string]interface{}
			for i2 := range txItem.ValuesBatch {
				values, err := raw2vals(txItem.ValuesBatch[i2])
				if err != nil {
					reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
					continue
				}

				if txItem.Encoder != nil {
					if err := encrypt(*txItem.Encoder, values); err != nil {
						reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
						continue
					}
				}

				valuesBatch = append(valuesBatch, values)
			}

			retE, err := processForExecBatch(tx, sqll, valuesBatch)
			if err != nil {
				reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
				continue
			}

			ret.Results[i] = *retE
		} else {
			// At most one values set (be it query or statement)
			values, err := raw2vals(txItem.Values)
			if err != nil {
				reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
				continue
			}

			if txItem.Encoder != nil {
				if err := encrypt(*txItem.Encoder, values); err != nil {
					reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
					continue
				}
			}

			if hasResultSet {
				// Query
				// Externalized in a func so that defer rows.Close() actually runs
				retWR, err := processWithResultSet(tx, sqll, txItem.Decoder, values)
				if err != nil {
					reportError(err, fiber.StatusInternalServerError, i, txItem.NoFail, ret.Results)
					continue
				}

				ret.Results[i] = *retWR
			} else {
				// Statement
				retE, err := processForExec(tx, sqll, values)
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
