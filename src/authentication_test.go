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
	"os"
	"testing"
	"time"

	"github.com/proofrock/ws4sql/structs"
	"github.com/proofrock/ws4sql/utils"
)

// Request Authentication ('INLINE' mode)

func TestSetupAuthCreds(t *testing.T) {
	os.Remove("../test/test0.db")
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")

	// test0 is not authenticated, test1 has structs.Credentials, test2 uses an auth query
	// init statements are also tested here
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test0"),
					Path:           utils.Ptr("../test/test0.db"),
					DisableWALMode: utils.Ptr(true),
				},
			},
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test1"),
					Path:           utils.Ptr("../test/test1.db"),
					DisableWALMode: utils.Ptr(true),
				},
				Auth: &structs.Authr{
					Mode: "INLINE",
					ByCredentials: []structs.CredentialsCfg{
						{
							User:     "pietro",
							Password: "hey",
						},
						{
							User:           "paolo",
							HashedPassword: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", // "ciao"
						},
					},
				},
			},
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test2"),
					Path:           utils.Ptr("../test/test2.db"),
					DisableWALMode: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE AUTH (USER TEXT PRIMARY KEY, PASS TEXT)",
					"INSERT INTO AUTH VALUES ('_pietro', 'hey'), ('_paolo', 'ciao')",
				},
				Auth: &structs.Authr{
					Mode:    "inline", // check if case insensitive
					ByQuery: "SELECT 1 FROM AUTH WHERE USER = :user AND PASS = :password",
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestNoAuthButAuthPassed(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "gigi",
			Password: "ciao",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test0", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestNoAuth1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestNoAuth2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestNoAuthWithCreds1(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "piero",
			Password: "hey",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestNoAuthWithCreds2(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "paolo",
			Password: "hey",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestAuthWithCreds1(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "pietro",
			Password: "hey",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestAuthWithCreds2(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "paolo",
			Password: "ciao",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestNoAuthWithQuery1(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "_piero",
			Password: "hey",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestNoAuthWithQuery2(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "_paolo",
			Password: "hey",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestAuthWithQuery1(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "_pietro",
			Password: "hey",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestAuthWithQuery2(t *testing.T) {
	req := structs.Request{
		Credentials: &structs.Credentials{
			User:     "_paolo",
			Password: "ciao",
		},
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestTeardownAuth(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
	os.Remove("../test/test0.db")
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")
}

// Basic Authentication ('HTTP' mode)

func TestBASetupAuthCreds(t *testing.T) {
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")

	// test1 has structs.Credentials, test2 uses an auth query
	// init statements are also tested here
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test1"),
					Path:           utils.Ptr("../test/test1.db"),
					DisableWALMode: utils.Ptr(true),
				},
				Auth: &structs.Authr{
					Mode: "HTTP",
					ByCredentials: []structs.CredentialsCfg{
						{
							User:     "pietro",
							Password: "hey",
						},
						{
							User:           "paolo",
							HashedPassword: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", // "ciao"
						},
					},
				},
			},
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test2"),
					Path:           utils.Ptr("../test/test2.db"),
					DisableWALMode: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE AUTH (USER TEXT PRIMARY KEY, PASS TEXT)",
					"INSERT INTO AUTH VALUES ('_pietro', 'hey'), ('_paolo', 'ciao')",
				},
				Auth: &structs.Authr{
					Mode:    "http", // check if case insensitive
					ByQuery: "SELECT 1 FROM AUTH WHERE USER = :user AND PASS = :password",
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestBANoAuth1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestBANoAuth2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestBANoAuthWithCreds1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test1", req, "piero", "hey", t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestBANoAuthWithCreds2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test1", req, "paolo", "hey", t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestBAAuthWithCreds1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test1", req, "pietro", "hey", t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestBAAuthWithCreds2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test1", req, "paolo", "ciao", t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestBANoAuthWithQuery1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test2", req, "_piero", "hey", t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestBANoAuthWithQuery2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test2", req, "_paolo", "hey", t)

	if code != 401 {
		t.Errorf("did not fail with 401: %s", body)
		return
	}
}

func TestBAAuthWithQuery1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test2", req, "_pietro", "hey", t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestBAAuthWithQuery2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := callBA("test2", req, "_paolo", "ciao", t)

	if code != 200 {
		t.Errorf("did not succeed, but should have: %s", body)
		return
	}
}

func TestBATeardownAuth(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")
}

func TestCustomCodeSetup(t *testing.T) {
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")

	errCode := 444

	// test1 has structs.Credentials, test2 uses an auth query
	// init statements are also tested here
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []structs.Db{
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test1"),
					Path:           utils.Ptr("../test/test1.db"),
					DisableWALMode: utils.Ptr(true),
				},
				Auth: &structs.Authr{
					Mode:            "HTTP",
					CustomErrorCode: &errCode,
					ByCredentials: []structs.CredentialsCfg{
						{
							User:     "pietro",
							Password: "hey",
						},
						{
							User:           "paolo",
							HashedPassword: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", // "ciao"
						},
					},
				},
			},
			{
				DatabaseDef: structs.DatabaseDef{
					Id:             utils.Ptr("test2"),
					Path:           utils.Ptr("../test/test2.db"),
					DisableWALMode: utils.Ptr(true),
				},
				InitStatements: []string{
					"CREATE TABLE AUTH (USER TEXT PRIMARY KEY, PASS TEXT)",
					"INSERT INTO AUTH VALUES ('_pietro', 'hey'), ('_paolo', 'ciao')",
				},
				Auth: &structs.Authr{
					Mode:            "inline", // check if case insensitive
					CustomErrorCode: &errCode,
					ByQuery:         "SELECT 1 FROM AUTH WHERE USER = :user AND PASS = :password",
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestCustomCodeNoAuth1(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test1", req, t)

	if code != 444 {
		t.Errorf("did not fail with 444: %s", body)
		return
	}
}

func TestCustomCodeNoAuth2(t *testing.T) {
	req := structs.Request{
		Transaction: []structs.RequestItem{
			{
				Query: "SELECT 1",
			},
		},
	}

	code, body, _ := call("test2", req, t)

	if code != 444 {
		t.Errorf("did not fail with 444: %s", body)
		return
	}
}

func TestCustomCodeTeardown(t *testing.T) {
	time.Sleep(time.Second)
	Shutdown()
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")
}
