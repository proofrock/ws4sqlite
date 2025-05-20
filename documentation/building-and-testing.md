# 🏗 Building & Testing

ws4sql is distributed with binaries for the major combinations of OSs and architectures. I'm trying to do portable builds as much as possible. For these reasons, building ws4sql is generally not required, but it could be useful to ensure that the binary matches the sources.

## Supported platforms

These are platforms for which we'll provide binaries at the time of the release.

| OS             | Arch         | Notes                                       |
| -------------- | ------------ | ------------------------------------------- |
| Linux          | amd64, arm64 | Static build for cross-distro compatibility |
| MacOS (darwin) | amd64, arm64 |                                             |
| Windows        | amd64        |                                             |

## Building

ws4sql is a Go(lang) program, that uses Go 1.24. The Go toolset makes compilation and cross-compiling very convenient, but there are some prerequisites.

* Go 1.24
* Make

I included a Make file to script the building under Linux, so if you have all the prerequisites it should be a matter of using:

```bash
git clone https://github.com/proofrock/ws4sql
cd ws4sql
make <build_target>
# You will find the binary in the bin/ directory.
```

And replace `make <build_target>` with one of the following:

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

## Testing

The code includes unit tests. Use the appropriate target for make:

```bash
make test
```

If you don't use this method, beware that a complete test that takes at least three minutes to complete (there are sleep's that ensure this); be sure to provide a suitable timeout value.
