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
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	mllog "github.com/proofrock/go-mylittlelogger"
	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

const (
	authModeInline = "INLINE"
	authModeHttp   = "HTTP"
)

// Checks auth. If auth is granted, returns nil, if not an error.
// Version with explicit credentials, called by the authentication
// middleware and by the "other" auth function, that accepts
// a request.
func applyAuthCreds(db *structs.Db, user, password string) error {
	if db.Auth.ByQuery != "" {
		// Auth via query. Looks into the database for the credentials;
		// needs a query that is correctly parametrized.
		nameds := utils.Vals2nameds(map[string]interface{}{"user": user, "password": password})
		row := db.DbConn.QueryRowContext(context.Background(), db.Auth.ByQuery, nameds...)
		var foo interface{}
		if err := row.Scan(&foo); err == sql.ErrNoRows {
			return errors.New("wrong credentials")
		} else if err != nil {
			return fmt.Errorf("in checking credentials: %s", err.Error())
		} else {
			return nil
		}
	} else {
		passedSHA := sha256.Sum256([]byte(password))
		expectedSHA, ok := db.Auth.HashedCreds[user]
		if !ok || !bytes.Equal(expectedSHA, passedSHA[:]) {
			return errors.New("wrong credentials")
		}
	}
	return nil
}

// Checks auth. If auth is granted, returns nil, if not an error.
// Version with request, extracts the credentials from the request
// (when authmode = INLINE) and delegates to applyAuthCreds()
func applyAuth(db *structs.Db, req *structs.Request) error {
	if req.Credentials == nil {
		return errors.New("missing auth credentials")
	}
	return applyAuthCreds(db, req.Credentials.User, req.Credentials.Password)
}

// Parses the authentication configurations. Builds a few structures,
// should be pretty straightforward to read.
func parseAuth(db *structs.Db) {
	auth := *db.Auth
	if strings.ToUpper(auth.Mode) != authModeInline && strings.ToUpper(auth.Mode) != authModeHttp {
		mllog.Fatal("Auth Mode must be INLINE or HTTP")
	}

	if (auth.ByCredentials == nil) == (auth.ByQuery == "") { // == is "NOT XOR"
		mllog.Fatal("one and only one of 'byQuery' and 'byCredentials' must be specified")
	}

	if auth.ByQuery != "" {
		if !strings.Contains(auth.ByQuery, ":user") || !strings.Contains(auth.ByQuery, ":password") {
			mllog.Fatal("byQuery: sql must include :user and :password named parameters")
		}
		mllog.StdOut("  + Authentication enabled, with query")
	} else {
		(*db).Auth.HashedCreds = make(map[string][]byte)
		for i := range auth.ByCredentials {
			if auth.ByCredentials[i].User == "" {
				mllog.Fatal("no user for credential")
			}
			var b []byte
			if (auth.ByCredentials[i].HashedPassword == "") == (auth.ByCredentials[i].Password == "") {
				mllog.Fatal("one and only one of 'password' and 'hashedPassword' must be specified")
			}
			// Converts all the password to hashes, if they weren't passed as hashes in the
			// first place. For uniformity and (vaguely) security.
			if auth.ByCredentials[i].HashedPassword != "" {
				var err error
				b, err = hex.DecodeString(auth.ByCredentials[i].HashedPassword)
				if err != nil || len(b) != 32 {
					mllog.Fatalf("for db '%s', hashedPassword doesn't seem to be SHA256/hex.", *db.DatabaseDef.Id)
				}
			} else {
				bytes32 := sha256.Sum256([]byte(auth.ByCredentials[i].Password))
				b = bytes32[:]
			}
			(*db).Auth.HashedCreds[auth.ByCredentials[i].User] = b
		}
		mllog.StdOutf("  + Authentication enabled, with %d credentials", len((*db).Auth.HashedCreds))
	}

	if auth.CustomErrorCode != nil {
		mllog.StdOutf("  + Custom code for Unauthorized: %d", *auth.CustomErrorCode)
	}
}
