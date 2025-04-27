.PHONY: test

cleanup:
	- rm -r bin
	- rm src/ws4sql
	- ls github.com && sudo rm -rf github.com # xgo build dir

upd-libraries:
	cd src; go get -u
	cd src; go mod tidy
	
test:
	cd src; go test -v -timeout 8m

build-prepare:
	make cleanup
	mkdir bin

build-nostatic:
	make build-prepare
	cd src; go build -trimpath
	mv src/ws4sql bin/

build-static-linux:
	make build-prepare
	cd src; VERSION="v0.0.999" bash build.sh LINUX
	mv src/ws4sql bin/

build-static-windows:
	make build-prepare
	cd src; VERSION="v0.0.999" bash build.sh WIN
	mv src/ws4sql.exe bin/

build-static-macos:
	make build-prepare
	cd src; VERSION="v0.0.999" bash build.sh MAC
	mv src/ws4sql bin/

# The following three targets are used in Github Actions
# Can be also run in an Alpine Linux context with the following packages
#   musl-dev go g++ make openssl openssl-dev openssl-libs-static zstd bash

build-static-ci-common:
	make build-prepare
	cd src && bash build.sh CI
	mv src/ws4sql bin/

build-static-ci-linux-musl-amd64:
	cp precompiled/libduckdb_bundle/linux-musl-amd64/libduckdb_bundle.a.zst src/
	make build-static-ci-common

build-static-ci-linux-musl-arm64:
	cp precompiled/libduckdb_bundle/linux-musl-arm64/libduckdb_bundle.a.zst src/
	make build-static-ci-common
