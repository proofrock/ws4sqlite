# Linux

To compile this prerequisite, set up an Alpine Linux (edge), for example in docker - and use this broad "recipe":

```bash
apk add --no-cache findutils musl-dev go g++ git make cmake ninja openssl openssl-dev openssl-libs-static python3 zstd
git clone -b "v1.1.3" https://github.com/duckdb/duckdb
cd duckdb
CFLAGS="-O3 -fPIC" CXXFLAGS="-O3 -fPIC" BUILD_SHELL=0 BUILD_UNITTESTS=0 DUCKDB_PLATFORM=any ENABLE_EXTENSION_AUTOLOADING=1 ENABLE_EXTENSION_AUTOINSTALL=1 BUILD_EXTENSIONS="json;httpfs;parquet" make bundle-library -j4
cp duckdb/build/release/libduckdb_bundle.a .
zstd -T4 -19 duckdb/build/release/libduckdb_bundle.a
```
