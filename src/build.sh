#!/bin/bash

# This script is used to build a binary

# Copy inside the src/ dir
# Env vars that must be already set: VERSION

case $1 in
    "MAC")
        go build -trimpath -a -o ws4sql \
            -tags="netgo osusergo sqlite_omit_load_extension" \
            -ldflags="-w -buildid=\"$VERSION\" -X \"main.version=$VERSION\""
        ;;
    "LINUX")
        go build -trimpath -a -o ws4sql \
            -tags="netgo osusergo sqlite_omit_load_extension" \
            -ldflags="-w -buildid=\"$VERSION\" -X \"main.version=$VERSION\" -extldflags \"-static\""
        ;;
    "WIN")
        go build -trimpath -a -o ws4sql.exe \
            -tags="netgo osusergo sqlite_omit_load_extension" \
            -ldflags="-w -buildid=\"$VERSION\" -X \"main.version=$VERSION\" -extldflags \"-static\""
        ;;
    "CI" )
        zstd -dk libduckdb_bundle.a.zst
        # '-O3' gives a warning in sqlite
        # Maybe -ldflags="buildid=\"$VERSION\" ..."
        CGO_ENABLED=1 \
        CPPFLAGS="-DDUCKDB_STATIC_BUILD" \
        CGO_CFLAGS="-O2" CGO_CXXFLAGS="-O2" \
        CGO_LDFLAGS="-static -lduckdb_bundle -lssl -lcrypto -lstdc++ -L." \
        go build \
          -buildvcs=false \
          -trimpath \
          -tags="netgo osusergo sqlite_omit_load_extension duckdb_use_static_lib" \
          -ldflags="-X \"main.version=$VERSION\"" \
          -o ws4sql
        rm libduckdb_bundle.a
        rm libduckdb_bundle.a.zst
esac

