.PHONY: test

build-prepare:
	make cleanup
	mkdir bin

cleanup:
	- rm -r bin
	- rm src/ws4sqlite

build:
	make build-prepare
	cd src; CGO_ENABLED=0 go build -a -tags netgo,osusergo,sqlite_omit_load_extension -ldflags '-w -extldflags "-static"' -o ws4sqlite -trimpath
	mv src/ws4sqlite bin/

zbuild:
	make build
	cd bin; 7zr a -mx9 -t7z ws4sqlite-v0.12.5-`uname -s|tr '[:upper:]' '[:lower:]'`-`uname -m`.7z ws4sqlite

build-nostatic:
	make build-prepare
	cd src; CGO_ENABLED=0 go build -o ws4sqlite -trimpath
	mv src/ws4sqlite bin/

zbuild-nostatic:
	make build-nostatic
	cd bin; 7zr a -mx9 -t7z ws4sqlite-v0.12.5-`uname -s|tr '[:upper:]' '[:lower:]'`-`uname -m`.7z ws4sqlite

zbuild-all:
	make build-prepare
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo,osusergo -ldflags '-w -extldflags "-static"' -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.12.5-linux-amd64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -a -tags netgo,osusergo -ldflags '-w -extldflags "-static"' -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.12.5-linux-arm.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -tags netgo,osusergo -ldflags '-w -extldflags "-static"' -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.12.5-linux-arm64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build -a -tags netgo,osusergo -ldflags '-w -extldflags "-static"' -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.12.5-linux-riscv64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.12.5-darwin-amd64.zip ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.12.5-darwin-arm64.zip ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.12.5-win-amd64.zip ws4sqlite.exe
	rm src/ws4sqlite.exe
	cd src; CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath
	cd src; zip -9 ../bin/ws4sqlite-v0.12.5-win-arm64.zip ws4sqlite.exe
	rm src/ws4sqlite.exe
	cd src; CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.12.5-freebsd-amd64.tar.gz ws4sqlite
	rm src/ws4sqlite
	cd src; CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -trimpath
	cd src; tar czf ../bin/ws4sqlite-v0.12.5-freebsd-arm64.tar.gz ws4sqlite
	rm src/ws4sqlite

test:
	cd src; go test -v -timeout 5m

docker:
	sudo docker build --no-cache -t local_ws4sqlite:latest .

docker-publish:
	## Prepare system with:
	## (verify which is latest at https://hub.docker.com/r/docker/binfmt/tags)
	sudo docker run --privileged --rm docker/binfmt:a7996909642ee92942dcd6cff44b9b95f08dad64
	sudo docker buildx build --no-cache -t germanorizzo/ws4sqlite:v0.12.5-amd64 .
	sudo docker buildx build --no-cache --platform linux/arm/v7 -t germanorizzo/ws4sqlite:v0.12.5-arm .
	sudo docker buildx build --no-cache --platform linux/arm64/v8 -t germanorizzo/ws4sqlite:v0.12.5-arm64 .
	sudo docker push germanorizzo/ws4sqlite:v0.12.5-amd64
	sudo docker push germanorizzo/ws4sqlite:v0.12.5-arm
	sudo docker push germanorizzo/ws4sqlite:v0.12.5-arm64
	sudo docker manifest create -a germanorizzo/ws4sqlite:v0.12.5 germanorizzo/ws4sqlite:v0.12.5-amd64 germanorizzo/ws4sqlite:v0.12.5-arm germanorizzo/ws4sqlite:v0.12.5-arm64
	sudo docker manifest push germanorizzo/ws4sqlite:v0.12.5
	sudo docker manifest rm germanorizzo/ws4sqlite:latest
	sudo docker manifest create germanorizzo/ws4sqlite:latest germanorizzo/ws4sqlite:v0.12.5-amd64 germanorizzo/ws4sqlite:v0.12.5-arm germanorizzo/ws4sqlite:v0.12.5-arm64
	sudo docker manifest push germanorizzo/ws4sqlite:latest
