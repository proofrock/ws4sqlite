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
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/proofrock/ws4sql/structs"
)

func TestFileServer(t *testing.T) {
	serveDir := "../test/"
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		ServeDir: &serveDir,
	}
	go launch(cfg, true)
	time.Sleep(time.Second)
	client := &fiber.Client{}
	get := client.Get("http://localhost:12321/mem1.yaml")

	code, _, errs := get.String()

	if len(errs) > 0 {
		t.Error(errs[0])
	}

	if code != 200 {
		t.Error("did not succeed")
	}

	time.Sleep(time.Second)

	Shutdown()
}

func TestFileServerKO(t *testing.T) {
	serveDir := "../test/"
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		ServeDir: &serveDir,
	}
	go launch(cfg, true)
	time.Sleep(time.Second)
	client := &fiber.Client{}
	get := client.Get("http://localhost:12321/mem1_nonexistent.yaml")

	code, _, errs := get.String()

	if len(errs) > 0 {
		t.Error(errs[0])
	}

	if code != 404 {
		t.Error("did not fail")
	}

	time.Sleep(time.Second)

	Shutdown()
}

func TestFileServerWithOverlap(t *testing.T) {
	serveDir := "../test/"
	cfg := structs.Config{
		Bindhost: "0.0.0.0",
		Port:     12321,
		ServeDir: &serveDir,
	}
	go launch(cfg, true)
	time.Sleep(time.Second)
	client := &fiber.Client{}
	get := client.Get("http://localhost:12321/test1")

	code, _, errs := get.String()

	if len(errs) > 0 {
		t.Error(errs[0])
	}

	if code != 200 {
		t.Error("did not succeed")
	}

	time.Sleep(time.Second)

	Shutdown()
}
