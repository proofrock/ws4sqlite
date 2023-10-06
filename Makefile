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
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-linux-amd64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-linux-arm.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-linux-arm64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-linux-riscv64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=s390x go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-linux-s390x.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.15.1-darwin-amd64.zip ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.15.1-darwin-arm64.zip ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.15.1-win-amd64.zip ws4sqlite.exe
	rm src/ws4sqlite.exe
	cd src; CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.15.1-win-arm64.zip ws4sqlite.exe
	rm src/ws4sqlite.exe
	cd src; CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-freebsd-amd64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.15.1-freebsd-arm64.tar.gz ws4sqlite
	rm src/ws4sqlite

test:
	cd src; go test -v -timeout 6m

docker:
	docker buildx build -f Dockerfile.containers --no-cache -t local_ws4sqlite:latest .

docker-multiarch:
	docker run --privileged --rm tonistiigi/binfmt --install arm64,arm
	docker buildx build -f Dockerfile.containers --no-cache -t germanorizzo/ws4sqlite:v0.15.1-amd64 .
	docker buildx build -f Dockerfile.containers --no-cache --platform linux/arm/v7 -t germanorizzo/ws4sqlite:v0.15.1-arm .
	docker buildx build -f Dockerfile.containers --no-cache --platform linux/arm64/v8 -t germanorizzo/ws4sqlite:v0.15.1-arm64 .

docker-publish:
	make docker-multiarch
	docker push germanorizzo/ws4sqlite:v0.15.1-amd64
	docker push germanorizzo/ws4sqlite:v0.15.1-arm
	docker push germanorizzo/ws4sqlite:v0.15.1-arm64
	docker manifest create -a germanorizzo/ws4sqlite:v0.15.1 germanorizzo/ws4sqlite:v0.15.1-amd64 germanorizzo/ws4sqlite:v0.15.1-arm germanorizzo/ws4sqlite:v0.15.1-arm64
	docker manifest push germanorizzo/ws4sqlite:v0.15.1
	- docker manifest rm germanorizzo/ws4sqlite:latest
	docker manifest create germanorizzo/ws4sqlite:latest germanorizzo/ws4sqlite:v0.15.1-amd64 germanorizzo/ws4sqlite:v0.15.1-arm germanorizzo/ws4sqlite:v0.15.1-arm64
	docker manifest push germanorizzo/ws4sqlite:latest

docker-devel:
	docker buildx build -f Dockerfile.containers --no-cache -t germanorizzo/ws4sqlite:edge .
	docker push germanorizzo/ws4sqlite:edge

docker-test-and-zbuild-all:
	docker buildx build -f Dockerfile.binaries --target export -t tmp_binaries_build . --output bin

docker-cleanup:
	docker builder prune -af
	docker image prune -af