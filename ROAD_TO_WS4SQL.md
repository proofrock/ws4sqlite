The next version of `ws4sqlite` will be called `ws4sql`, because it will support more RDBMs than sqlite.

The version in this branch is a work in progress to slowly add features and (unfortunately) breaking changes; here is a review of what changed compared to `ws4sqlite` "stable" + a migration path.

# Changes

- SQLite is embedded via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) and CGO. Should be way faster.
- Support for DuckDB (see below).
- Target platforms (because of CGO) are now 6 (`win/amd64`, `macos/amd64`, `macos/arm64`, `linux/amd64`, `linux/arm64`, `linux/arm6`).
- [**BREAKING CHANGE**] When running the app, the config files must be specified on the command line, the file paths cannot be used anymore (there). This is described in the "Migration" section below. The file path is in the config file.
- The only exception is a "simple case" to serve a file path without any config. This can be done with the new `--quick-db` parameter.
- Fail fast if the request is empty, don't even attempt to authenticate

# Migration

- For any `--db` and `--mem-db` switch that was used, an explicit YAML config file must be created. The format is the same, but there is a new section at the beginning:

```yaml
database:
  type: SQLITE          # SQLITE or DUCKDB. If omitted, defaults to SQLITE      
  inMemory: false       # If type = SQLITE|DUCKDB. The db is a memory one? If omitted, defaults to false
  path: ".../test.db"   # If type = SQLITE|DUCKDB. The db file path.
  id: test              # If omitted and !inMemory, calculates it from the file name (if type = SQLITE|DUCKDB)
  disableWALMode: false # If type = SQLITE. Same as before, but moved here.
  readOnly: false       # Same as before, but moved here.
```

# Specific to DuckDB

- `noFail` is not supported.
- Placeholders for named parameters are `$VAL`, not `:VAL` as in SQLite.
- As DuckDB does not support read-only transactions, when `readOnly` is specified the requests won't be processed in a transaction.

# Roadmap

1. Support mariadb/mysql
1. Support postgresql
1. ...
1. Profit!
