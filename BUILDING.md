# How to build ws4sql

The build system uses `make`.

## Targets (normal builds)

### `make build-nostatic`

Builds a dinamically linked binary under the current architecture/os, in the `bin/` folder.

### `make build-static-linux`

Builds an almost-statically linked binary for Linux, under the current architecture, in the `bin/` folder.
It's not fully static, as duckdb is a bit tricky to compile statically. The CI does this using `musl` (see
below).

### `make build-static-windows`

Builds a statically linked binary for Windows, under the current architecture, in the `bin/` folder.

### `make build-static-macos`

Builds a somewhat-statically linked binary for Mac OS, under the current architecture, in the `bin/` folder.
It's not fully static, as Mac OS doesn't allow statically linking against OS libraries.

## Targets (CI builds)

The Github Actions script (`.github/workflows/main.yml`) compiles the linux binaries fully statically,
in an Alpine linux chroot, and using a precompiled version of the `libduckdb_bundle.a` that is static and
platform-independent (`-fPIC`). If you have an Alpine Linux environment, you can use the following targets.

- `build-static-ci-linux-musl-amd64`
- `build-static-ci-linux-musl-arm64`

Required packages are `musl-dev go g++ make openssl openssl-dev openssl-libs-static zstd`.

## Docker

The provided `Dockerfile` assumes that a `ws4sql` static binary is in the `bin/` directory.
