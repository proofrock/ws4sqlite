# How to build ws4sqlite

The build system uses `make`. There are two kinds of targets:

- "direct" builds, that use go(lang) tooling to build a statically or dinamically linked binary or set of binaries;
- docker-based builds, that build binaries or containers under a docker environment, using the "official" golang docker 
  image as a base.

All binaries generated for distribution are statically linked.

## Direct targets

#### make build

Builds a statically linked binary under the current architecture, in the `bin/` folder.

#### make build-nostatic

Builds a dinamically linked binary under the current architecture, in the `bin/` folder.

#### make zbuild-all

Builds statically linked binaries for all the supported OSs/architectures, also creating the distribution archives in
the `bin/` folder.

## Docker targets

In general, docker images are built in the debian-based official docker image; the generated binary is then distributed
in a `distroless/static-debian11` image. The final size is about 20Mb.

*NB: buildx must be installed/enabled*
*NB2: the **current** sources will be copied to the docker context.*

#### make docker-test-and-zbuild-all

Builds the distribution archives for the supported OSs/architectures (see `zbuild-all`) under a docker environment, and
copies them in the `bin/` folder.

#### make docker

Builds a docker image (called `local_ws4sqlite:latest`) in the current architecture.

#### make docker-multiarch

Builds docker images for AMD64, ARMv7 and ARM64v8. The images are named like the official ones, i.e. 
`germanorizzo/ws4sqlite:v0.xx.xx-xxx`.

#### make docker-multiarch
#### make docker-devel

Reserved, for publishing in Docker Hub.
