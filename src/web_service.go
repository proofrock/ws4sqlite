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
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/proofrock/crypgo"
	"github.com/wI2L/jettison"
)

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
// or just report an error in the single query
func reportError(err error, code int, reqIdx int, noFail bool, results []responseItem) []responseItem {
	if !noFail {
		panic(newWSError(reqIdx, code, err.Error()))
	}
	return append(results, ResItem4Error(capitalize(err.Error())))
}

func processWithResultSet(tx *sql.Tx, q string, decoder *requestItemCrypto, values map[string]interface{}) (responseItem, error) {
	resultSet := make([]map[string]interface{}, 0)

	rows, err := tx.Query(q, vals2nameds(values)...)
	if err != nil {
		return ResItemEmpty(), err
	}
	defer rows.Close()

	fields, err := rows.Columns()
	if err != nil {
		return ResItemEmpty(), err
	}
	for rows.Next() {
		values := make([]interface{}, len(fields))
		scans := make([]interface{}, len(fields))
		for i := range values {
			scans[i] = &values[i]
		}
		if err = rows.Scan(scans...); err != nil {
			return ResItemEmpty(), err
		}

		toAdd := make(map[string]interface{})
		for i := range values {
			toAdd[fields[i]] = values[i]
		}

		if decoder != nil {
			if err := decrypt(*decoder, toAdd); err != nil {
				return ResItemEmpty(), err
			}
		}
		resultSet = append(resultSet, toAdd)
	}

	if err = rows.Err(); err != nil {
		return ResItemEmpty(), err
	}

	return ResItem4Query(resultSet), nil
}

func processForExec(tx *sql.Tx, q string, values map[string]interface{}) (responseItem, error) {
	qres, err := tx.Exec(q, vals2nameds(values)...)
	if err != nil {
		return ResItemEmpty(), err
	}

	rAff, err := qres.RowsAffected()
	if err != nil {
		return ResItemEmpty(), err
	}

	return ResItem4Statement(rAff), nil
}

func processForExecBatch(tx *sql.Tx, q string, valuesBatch []map[string]interface{}) (responseItem, error) {
	ps, err := tx.Prepare(q)
	if err != nil {
		return ResItemEmpty(), err
	}
	defer ps.Close()

	var rAffs []int64
	for i := range valuesBatch {
		qres, err := ps.Exec(vals2nameds(valuesBatch[i])...)
		if err != nil {
			return ResItemEmpty(), err
		}

		rAff, err := qres.RowsAffected()
		if err != nil {
			return ResItemEmpty(), err
		}

		rAffs = append(rAffs, rAff)
	}

	return ResItem4Batch(rAffs), nil
}

func handler(c *fiber.Ctx) error {
	var body request
	if err := c.BodyParser(&body); err != nil {
		panic(newWSError(-1, fiber.StatusBadRequest, "in parsing body: %s", err.Error()))
	}

	databaseId := c.Params("databaseId")
	if databaseId == "" {
		panic(newWSError(-1, fiber.StatusNotFound, "missing database ID"))
	}

	_db, found := dbs[databaseId]
	if !found {
		panic(newWSError(-1, fiber.StatusNotFound, "database with ID '%s' not found", databaseId))
	}
	db := _db

	if db.Auth != nil && strings.ToUpper(db.Auth.Mode) == authModeInline {
		if err := applyAuth(&db, &body); err != nil {
			db.Mutex.Lock() // When unauthenticated waits for 2s, and doesn't parallelize, to hinder brute force attacks
			time.Sleep(2 * time.Second)
			db.Mutex.Unlock()
			panic(newWSError(-1, fiber.StatusUnauthorized, err.Error()))
		}
	}

	if body.Transaction == nil || len(body.Transaction) == 0 {
		panic(newWSError(-1, fiber.StatusBadRequest, "missing statements list ('transaction' node)"))
	}

	var ret response

	dbc, err := db.Db.Conn(context.Background())
	if err != nil {
		panic(newWSError(-1, fiber.StatusInternalServerError, err.Error()))
	}
	defer dbc.Close()

	tx, err := dbc.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: db.ReadOnly})
	if err != nil {
		panic(newWSError(-1, fiber.StatusInternalServerError, err.Error()))
	}

	tainted := true // if I reach the end of the method, I switch this to false to signal success
	defer func() {
		if tainted {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for i := range body.Transaction {
		if (body.Transaction[i].Query == "") == (body.Transaction[i].Statement == "") { // both null or both populated
			ret.Results = reportError(errors.New("one and only one of query or statement must be provided"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		hasResultSet := body.Transaction[i].Query != ""

		if hasResultSet && body.Transaction[i].Encoder != nil {
			ret.Results = reportError(errors.New("cannot specify an encoder for a query"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
		}

		if !hasResultSet && body.Transaction[i].Decoder != nil {
			ret.Results = reportError(errors.New("cannot specify a decoder for a statement"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
		}

		if !hasResultSet {
			body.Transaction[i].Query = body.Transaction[i].Statement
		}

		if len(body.Transaction[i].Values) != 0 && len(body.Transaction[i].ValuesBatch) != 0 {
			ret.Results = reportError(errors.New("cannot specify both values and valuesBatch"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		if hasResultSet && len(body.Transaction[i].ValuesBatch) != 0 {
			ret.Results = reportError(errors.New("cannot specify valuesBatch for queries (only for statements)"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
			continue
		}

		hasBatch := len(body.Transaction[i].ValuesBatch) != 0

		var q string
		if strings.HasPrefix(body.Transaction[i].Query, "#") {
			var ok bool
			q, ok = db.StoredStatsMap[body.Transaction[i].Query[1:]]
			if !ok {
				ret.Results = reportError(errors.New("a stored statement is required, but did not find it"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}
		} else {
			if db.UseOnlyStoredStatements {
				ret.Results = reportError(errors.New("configured to serve only stored statements, but SQL is passed"), fiber.StatusBadRequest, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}
			q = body.Transaction[i].Query
		}

		if hasBatch {
			var valuesBatch []map[string]interface{}
			for i2 := range body.Transaction[i].ValuesBatch {
				values, err := raw2vals(body.Transaction[i].ValuesBatch[i2])
				if err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}

				if body.Transaction[i].Encoder != nil {
					if err := encrypt(*body.Transaction[i].Encoder, values); err != nil {
						ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
						continue
					}
				}

				valuesBatch = append(valuesBatch, values)
			}

			retE, err := processForExecBatch(tx, q, valuesBatch)
			if err != nil {
				ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}

			ret.Results = append(ret.Results, retE)
		} else {
			values, err := raw2vals(body.Transaction[i].Values)
			if err != nil {
				ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
				continue
			}

			if body.Transaction[i].Encoder != nil {
				if err := encrypt(*body.Transaction[i].Encoder, values); err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}
			}

			if hasResultSet {
				// Externalized in a func so that defer rows.Close() actually runs
				retWR, err := processWithResultSet(tx, q, body.Transaction[i].Decoder, values)
				if err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}

				ret.Results = append(ret.Results, retWR)
			} else {
				retE, err := processForExec(tx, q, values)
				if err != nil {
					ret.Results = reportError(err, fiber.StatusInternalServerError, i, body.Transaction[i].NoFail, ret.Results)
					continue
				}

				ret.Results = append(ret.Results, retE)
			}
		}
	}

	bytes, err := jettison.Marshal(ret)
	if err != nil {
		panic(newWSError(-1, fiber.StatusInternalServerError, err.Error()))
	} else {
		tainted = false
	}

	c.Response().Header.Add("Content-Type", "application/json")

	return c.Send(bytes)
}
