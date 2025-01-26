.PHONY: test

build-prepare:
	make cleanup
	mkdir bin

cleanup:
	- rm -r bin
	- rm src/ws4sql
	- ls github.com && sudo rm -rf github.com # xgo build dir

upd-libraries:
	cd src; go get -u
	cd src; go mod tidy

build:
	make build-prepare
	cd src; go build -trimpath
	mv src/ws4sql bin/

build-windows:
	make build-prepare
	cd src; go build -trimpath
	mv src/ws4sql.exe bin/

build-static:
	make build-prepare
	cd src; go build -trimpath -a -tags="netgo osusergo sqlite_omit_load_extension" -ldflags='-w -extldflags "-static"'
	mv src/ws4sql bin/

build-static-linux-musl-amd64:
	# In an Alpine Linux context with the following packages
	#   musl-dev go g++ make openssl openssl-dev openssl-libs-static zstd
	make build-prepare
	cp precompiled/libduckdb_bundle/linux-musl-amd64/libduckdb_bundle.a.zst src/
	cd src && zstd -d libduckdb_bundle.a.zst
	cd src && CGO_CFLAGS="-O2 -fPIC" CGO_CXXFLAGS="-O2 -fPIC" CGO_LDFLAGS="-lduckdb_bundle -lssl -lcrypto -L./" go build -buildvcs=false -trimpath -a -tags="netgo osusergo sqlite_omit_load_extension duckdb_use_static_lib" -ldflags='-w -extldflags "-static"'
	mv src/ws4sql bin/

build-static-linux-musl-arm64:
	# In an Alpine Linux context with the following packages
	#   musl-dev go g++ make openssl openssl-dev openssl-libs-static zstd
	make build-prepare
	cp precompiled/libduckdb_bundle/linux-musl-arm64/libduckdb_bundle.a.zst src/
	cd src && zstd -d libduckdb_bundle.a.zst
	cd src && CGO_CFLAGS="-O2 -fPIC" CGO_CXXFLAGS="-O2 -fPIC" CGO_LDFLAGS="-lduckdb_bundle -lssl -lcrypto -L./" go build -buildvcs=false -trimpath -a -tags="netgo osusergo sqlite_omit_load_extension duckdb_use_static_lib" -ldflags='-w -extldflags "-static"'
	mv src/ws4sql bin/

test:
	cd src; go test -v -timeout 6m
