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

dist-pre:
	docker pull techknowlogick/xgo:latest
	GOBIN=/home/devel/local/bin/ go install src.techknowlogick.com/xgo@latest

dist:
	xgo -gcflags='-trimpath -a' -tags="netgo osusergo sqlite_omit_load_extension" -ldflags='-w -extldflags "-static"' --targets=linux/amd64,linux/arm-6,linux/arm64 ./src/
	xgo -trimpath --targets=windows-10.0/amd64,darwin/* ./src/
	sudo chown -R `stat -c "%U:%G" Makefile` "github.com"
	mv github.com/proofrock/ bin
	rm -rf github.com/
	mv bin/ws4sql-windows-10.0-amd64.exe bin/ws4sql.exe
	cd bin/ && zip -9 ws4sql-v0.17dev3-win-x86_64.zip ws4sql.exe
	cd bin/ && gpg --sign --default-key oss@germanorizzo.it --output ws4sql-v0.17dev3-win-x86_64.zip.gpg --detach-sig ws4sql-v0.17dev3-win-x86_64.zip
	rm bin/ws4sql.exe
	mv bin/ws4sql-darwin-10.12-amd64 bin/ws4sql
	cd bin/ && zip -9 ws4sql-v0.17dev3-darwin-x86_64.zip ws4sql
	cd bin/ && gpg --sign --default-key oss@germanorizzo.it --output ws4sql-v0.17dev3-darwin-x86_64.zip.gpg --detach-sig ws4sql-v0.17dev3-darwin-x86_64.zip
	mv bin/ws4sql-darwin-10.12-arm64 bin/ws4sql
	cd bin/ && zip -9 ws4sql-v0.17dev3-darwin-arm64.zip ws4sql
	cd bin/ && gpg --sign --default-key oss@germanorizzo.it --output ws4sql-v0.17dev3-darwin-arm64.zip.gpg --detach-sig ws4sql-v0.17dev3-darwin-arm64.zip
	mv bin/ws4sql-linux-amd64 bin/ws4sql
	cd bin/ && bash -c "tar c ws4sql | gzip -9 > ws4sql-v0.17dev3-linux-x86_64.tar.gz"
	cd bin/ && gpg --sign --default-key oss@germanorizzo.it --output ws4sql-v0.17dev3-linux-x86_64.tar.gz.gpg --detach-sig ws4sql-v0.17dev3-linux-x86_64.tar.gz
	mv bin/ws4sql-linux-arm64 bin/ws4sql
	cd bin/ && bash -c "tar c ws4sql | gzip -9 > ws4sql-v0.17dev3-linux-arm64.tar.gz"
	cd bin/ && gpg --sign --default-key oss@germanorizzo.it --output ws4sql-v0.17dev3-linux-arm64.tar.gz.gpg --detach-sig ws4sql-v0.17dev3-linux-arm64.tar.gz
	mv bin/ws4sql-linux-arm-6 bin/ws4sql
	cd bin/ && bash -c "tar c ws4sql | gzip -9 > ws4sql-v0.17dev3-linux-armv6.tar.gz"
	cd bin/ && gpg --sign --default-key oss@germanorizzo.it --output ws4sql-v0.17dev3-linux-armv6.tar.gz.gpg --detach-sig ws4sql-v0.17dev3-linux-armv6.tar.gz
	rm bin/ws4sql

docker:
	docker buildx build -f Dockerfile.x86_64 --no-cache -t local_ws4sql:latest .

docker-multiarch:
	docker run --privileged --rm tonistiigi/binfmt --install arm64,arm
	docker buildx build -f Dockerfile.x86_64 --no-cache -t germanorizzo/ws4sql:v0.17dev3-amd64 .
	docker buildx build -f Dockerfile.arm64 --no-cache --platform linux/arm64/v8 -t germanorizzo/ws4sql:v0.17dev3-arm64 .

docker-publish:
	make docker-multiarch
	docker push germanorizzo/ws4sql:v0.17dev3-amd64
	docker push germanorizzo/ws4sql:v0.17dev3-arm64
	docker manifest create -a germanorizzo/ws4sql:v0.17dev3 germanorizzo/ws4sql:v0.17dev3-amd64 germanorizzo/ws4sql:v0.17dev3-arm64
	docker manifest push germanorizzo/ws4sql:v0.17dev3
	- docker manifest rm germanorizzo/ws4sql:latest
	docker manifest create germanorizzo/ws4sql:latest germanorizzo/ws4sql:v0.17dev3-amd64 germanorizzo/ws4sql:v0.17dev3-arm64
	docker manifest push germanorizzo/ws4sql:latest

docker-devel:
	docker buildx build -f Dockerfile.x86_64 --no-cache -t germanorizzo/ws4sql:edge .
	docker push germanorizzo/ws4sql:edge

docker-cleanup:
	docker builder prune -af
	docker image prune -af