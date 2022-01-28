package main

import (
	"os"
	"testing"
	"time"
)

// Request Authentication ('INLINE' mode)

func TestSetupAuthCreds(t *testing.T) {
	os.Remove("../test/test0.db")
	os.Remove("../test/test1.db")
	os.Remove("../test/test2.db")

	// test0 is not authenticated, test1 has credentials, test2 uses an auth query
	// init statements are also tested here
	cfg := config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []db{
			{
				Id:             "test0",
				Path:           "../test/test0.db",
				DisableWALMode: true,
			},
			{
				Id:             "test1",
				Path:           "../test/test1.db",
				DisableWALMode: true,
				Auth: &authr{
					Mode: "INLINE",
					ByCredentials: []credentialsCfg{
						{
							User: "pietro",
							Pass: "hey",
						},
						{
							User:       "paolo",
							HashedPass: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", // "ciao"
						},
					},
				},
			},
			{
				Id:             "test2",
				Path:           "../test/test2.db",
				DisableWALMode: true,
				InitStatements: []string{
					"CREATE TABLE AUTH (USER TEXT PRIMARY KEY, PASS TEXT)",
					"INSERT INTO AUTH VALUES ('_pietro', 'hey'), ('_paolo', 'ciao')",
				},
				Auth: &authr{
					Mode:    "inline", // check if case insensitive
					ByQuery: "SELECT 1 FROM AUTH WHERE USER = :user AND PASS = :pass",
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestNoAuthButAuthPassed(t *testing.T) {
	req := request{
		Credentials: &credentials{
			User: "gigi",
			Pass: "ciao",
		},
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "piero",
			Pass: "hey",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "paolo",
			Pass: "hey",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "pietro",
			Pass: "hey",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "paolo",
			Pass: "ciao",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "_piero",
			Pass: "hey",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "_paolo",
			Pass: "hey",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "_pietro",
			Pass: "hey",
		},
		Transaction: []requestItem{
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
	req := request{
		Credentials: &credentials{
			User: "_paolo",
			Pass: "ciao",
		},
		Transaction: []requestItem{
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

	// test1 has credentials, test2 uses an auth query
	// init statements are also tested here
	cfg := config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		Databases: []db{
			{
				Id:             "test1",
				Path:           "../test/test1.db",
				DisableWALMode: true,
				Auth: &authr{
					Mode: "HTTP",
					ByCredentials: []credentialsCfg{
						{
							User: "pietro",
							Pass: "hey",
						},
						{
							User:       "paolo",
							HashedPass: "b133a0c0e9bee3be20163d2ad31d6248db292aa6dcb1ee087a2aa50e0fc75ae2", // "ciao"
						},
					},
				},
			},
			{
				Id:             "test2",
				Path:           "../test/test2.db",
				DisableWALMode: true,
				InitStatements: []string{
					"CREATE TABLE AUTH (USER TEXT PRIMARY KEY, PASS TEXT)",
					"INSERT INTO AUTH VALUES ('_pietro', 'hey'), ('_paolo', 'ciao')",
				},
				Auth: &authr{
					Mode:    "http", // check if case insensitive
					ByQuery: "SELECT 1 FROM AUTH WHERE USER = :user AND PASS = :pass",
				},
			},
		},
	}
	go launch(cfg, true)

	time.Sleep(time.Second)
}

func TestBANoAuth1(t *testing.T) {
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
	req := request{
		Transaction: []requestItem{
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
