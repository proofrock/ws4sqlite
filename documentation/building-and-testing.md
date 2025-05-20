# 🏗 Building & Testing

ws4sql is distributed with binaries for the major combinations of OSs and architectures. I'm trying to do portable builds as much as possible. For these reasons, building ws4sql is generally not required, but it could be useful to ensure that the binary matches the sources.

### Supported platforms

These are platforms for which we'll provide binaries at the time of the release.

| OS             | Arch                       | Notes                                       |
| -------------- | -------------------------- | ------------------------------------------- |
| Linux          | amd64, arm, arm64, riscv64 | Static build for cross-distro compatibility |
| MacOS (darwin) | amd64, arm64               |                                             |
| Windows        | amd64, arm64               |                                             |
| FreeBSD        | amd64                      |                                             |

### Building

ws4sql is a Go(lang) program, that uses Go 1.18. The Go toolset makes compilation and cross-compiling very convenient, but there are some prerequisites.

* Go 1.17
* Make

I included a Make file to script the building under Linux, so if you have all the prerequisites it should be a matter of:

```bash
git clone https://github.com/proofrock/ws4sql
cd ws4sql
make build
# You will find the binary in the bin/ directory.
```

For MacOS replace Line 3 with:

```bash
make build-nostatic
```

For Windows, instead of the step at Line 3 do the following. I don't support using `make` under windows, though it could be feasible.

```bash
cd src
go build
```

### Testing

The code includes unit tests. Use the appropriate target for make:

```bash
make test
```

If you don't use this method, beware that a complete test that takes at least three minutes to complete (there are sleep's that ensure this); be sure to provide a suitable timeout value.
