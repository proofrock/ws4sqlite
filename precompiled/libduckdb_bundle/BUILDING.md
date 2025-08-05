# Linux

To compile this prerequisite, set up an Alpine Linux (edge), for example in docker - and use this broad "recipe":

```bash
# For example, install https://github.com/alpinelinux/alpine-chroot-install
# sudo ./alpine-chroot-install -b edge [-a aarch64] -d /home/mano/tmp/alpine -i /home/mano/Devel/ws4sqlite -t /tmp/
# /home/mano/tmp/alpine/enter-chroot
# cd
# /home/mano/tmp/alpine/destroy --remove

apk add --no-cache findutils musl-dev go g++ git make cmake ninja openssl openssl-dev openssl-libs-static python3 zstd
git clone -b "v1.3.2" https://github.com/duckdb/duckdb
cd duckdb
CFLAGS="-O2" CXXFLAGS="-O2" \
 DUCKDB_PLATFORM=linux \
 BUILD_SHELL=0 BUILD_UNITTESTS=0 \
 BUILD_EXTENSIONS="json;parquet" \
 ENABLE_EXTENSION_AUTOLOADING=1 \
 ENABLE_EXTENSION_AUTOINSTALL=1 \
 make bundle-library -j16
zstd -T16 -19 build/release/libduckdb_bundle.a
# get build/release/libduckdb_bundle.a.zst
```
cp build/release/libduckdb_bundle.a.zst /home/mano/Devel/ws4sqlite/precompiled/libduckdb_bundle/linux-musl-arm64/

# '-O3' gives a warning in sqlite
# CGO_ENABLED=1 CPPFLAGS="-DDUCKDB_STATIC_BUILD" CGO_CFLAGS="-O2 -fPIC" CGO_CXXFLAGS="-O2 -fPIC" CGO_LDFLAGS="-lduckdb_bundle -lssl -lcrypto -lstdc++ -L. -static" go build -buildvcs=false -trimpath -tags="netgo osusergo sqlite_omit_load_extension duckdb_use_static_lib" -ldflags="-buildid=\"$VERSION\" -X \"main.version=$VERSION\"" -o ws4sql

# Maybe -ldflags="buildid=\"$VERSION\" -X \"main.version=$VERSION\"" \

CGO_ENABLED=1 \
 CPPFLAGS="-DDUCKDB_STATIC_BUILD" \
 CGO_CFLAGS="-O2" \
 CGO_CXXFLAGS="-O2" \
 CGO_LDFLAGS="-static -lduckdb_bundle -lssl -lcrypto -lstdc++ -L." \
 go build \
  -buildvcs=false \
  -trimpath \
  -tags="netgo osusergo sqlite_omit_load_extension duckdb_use_static_lib" \
  -ldflags="-X \"main.version=$VERSION\"" \
  -o ws4sql
