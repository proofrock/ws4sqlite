# How to build ws4sql

The build system uses `make`. There are two kinds of targets:

- "direct" builds, that use go(lang) tooling and [xgo](https://github.com/techknowlogick/xgo) to build a statically or dinamically linked binary or set of binaries;
- docker image builds, that build docker images.

All linux binaries generated for distribution are statically linked. MacOS and Windows binaries are dynamically linked.

## Direct targets

#### make build

Builds a dinamically linked binary under the current architecture, in the `bin/` folder.

#### make build-static

Builds a statically linked binary under the current architecture, in the `bin/` folder.

#### make dist*:

Builds binaries for the 6 supported OSs/architectures, also creating the distribution archives in
the `bin/` folder. Uses [xgo](https://github.com/techknowlogick/xgo) for cross compiling.

It's actually a three-stage process: the first time use `make dist-pre` to setup the environment;
then `make dist` to build the binaries. Then you need to re-own the output dir; run the command
outputted by the last `make dist`. Finally, do `make dist-post`.

## Docker targets

The docker files assume that the `make dist*` stuff described above was performed.

Docker images are based on the `debian:stable-slim` official docker image.

*NB: buildx must be installed/enabled*

#### make docker

Builds a docker image (tagged `local_ws4sql:latest`) in the current architecture.

#### make docker-multiarch

Builds docker images for AMD64 and ARM64v8. The images are named like the official ones, i.e. 
`germanorizzo/ws4sql:v0.xx.xx-xxx`.

#### make docker-publish
#### make docker-devel

Reserved, for publishing in Docker Hub.
