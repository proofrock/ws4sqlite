.PHONY: test

build-prepare:
	make cleanup
	mkdir bin

cleanup:
	- rm -r bin
	- rm src/ws4sqlite

upd-libraries:
	cd src; go get -u
	cd src; go mod tidy

build:
	make build-prepare
	cd src; CGO_ENABLED=0 go build -trimpath
	mv src/ws4sqlite bin/

build-nostatic:
	make build-prepare
	cd src; go build -trimpath
	mv src/ws4sqlite bin/

zbuild-all:
	make build-prepare
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-linux-amd64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-linux-arm.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-linux-arm64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-linux-riscv64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=s390x go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-linux-s390x.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.0.0-darwin-amd64.zip ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.0.0-darwin-arm64.zip ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.0.0-win-amd64.zip ws4sqlite.exe
	rm src/ws4sqlite.exe
	cd src; CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.0.0-win-arm64.zip ws4sqlite.exe
	rm src/ws4sqlite.exe
	cd src; CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-freebsd-amd64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.0.0-freebsd-arm64.tar.gz ws4sqlite
	rm src/ws4sqlite

test:
	cd src; go test -v -timeout 6m

zxbuild-pre:
	docker pull techknowlogick/xgo:latest
	GOBIN=/home/devel/local/bin/ go install src.techknowlogick.com/xgo@latest

zxbuild:
	xgo -gcflags='-trimpath -a' -tags="netgo osusergo sqlite_omit_load_extension" -ldflags='-w -extldflags "-static"' --targets=linux/amd64,linux/arm-6,linux/arm64 ./src/
	xgo -trimpath --targets=windows-10.0/amd64,darwin/* ./src/
	echo "Run 'sudo chown -R $(whoami):$(whoami) \"github.com\" && make zxbuild-post'"

zxbuild-post:
	mv github.com/proofrock/ bin
	rm -rf github.com/
	mv bin/ws4sqlite-windows-10.0-amd64.exe bin/ws4slite.exe
	cd bin/ && zip -9 ws4sqlite-v0.0.0-win-x86_64.zip ws4slite.exe
	rm bin/ws4slite.exe
	mv bin/ws4sqlite-darwin-10.12-amd64 bin/ws4slite
	cd bin/ && bash -c "tar c ws4slite | gzip -9 > ws4sqlite-v0.0.0-darwin-x86_64.tar.gz"
	mv bin/ws4sqlite-darwin-10.12-arm64 bin/ws4slite
	cd bin/ && bash -c "tar c ws4slite | gzip -9 > ws4sqlite-v0.0.0-darwin-arm64.tar.gz"
	mv bin/ws4sqlite-linux-amd64 bin/ws4slite
	cd bin/ && bash -c "tar c ws4slite | gzip -9 > ws4sqlite-v0.0.0-linux-x86_64.tar.gz"
	mv bin/ws4sqlite-linux-arm64 bin/ws4slite
	cd bin/ && bash -c "tar c ws4slite | gzip -9 > ws4sqlite-v0.0.0-linux-arm64.tar.gz"
	mv bin/ws4sqlite-linux-arm-6 bin/ws4slite
	cd bin/ && bash -c "tar c ws4slite | gzip -9 > ws4sqlite-v0.0.0-linux-arm-v6.tar.gz"
	rm bin/ws4slite

docker:
	docker buildx build -f Dockerfile --no-cache -t local_ws4sqlite:latest .

docker-multiarch:
	docker run --privileged --rm tonistiigi/binfmt --install arm64,arm
	docker buildx build -f Dockerfile --no-cache -t germanorizzo/ws4sqlite:v0.0.0-amd64 .
	docker buildx build -f Dockerfile --no-cache --platform linux/arm/v7 -t germanorizzo/ws4sqlite:v0.0.0-arm .
	docker buildx build -f Dockerfile --no-cache --platform linux/arm64/v8 -t germanorizzo/ws4sqlite:v0.0.0-arm64 .

docker-publish:
	make docker-multiarch
	docker push germanorizzo/ws4sqlite:v0.0.0-amd64
	docker push germanorizzo/ws4sqlite:v0.0.0-arm
	docker push germanorizzo/ws4sqlite:v0.0.0-arm64
	docker manifest create -a germanorizzo/ws4sqlite:v0.0.0 germanorizzo/ws4sqlite:v0.0.0-amd64 germanorizzo/ws4sqlite:v0.0.0-arm germanorizzo/ws4sqlite:v0.0.0-arm64
	docker manifest push germanorizzo/ws4sqlite:v0.0.0
	- docker manifest rm germanorizzo/ws4sqlite:latest
	docker manifest create germanorizzo/ws4sqlite:latest germanorizzo/ws4sqlite:v0.0.0-amd64 germanorizzo/ws4sqlite:v0.0.0-arm germanorizzo/ws4sqlite:v0.0.0-arm64
	docker manifest push germanorizzo/ws4sqlite:latest

docker-devel:
	docker buildx build -f Dockerfile --no-cache -t germanorizzo/ws4sqlite:edge .
	docker push germanorizzo/ws4sqlite:edge

docker-test-and-zbuild-all:
	docker buildx build -f Dockerfile.binaries --target export -t tmp_binaries_build . --output bin

docker-cleanup:
	docker builder prune -af
	docker image prune -af