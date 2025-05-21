# Road to `ws4sql`

The next version of `ws4sqlite` will be called `ws4sql`, because it will support more RDBMs than sqlite.

The version in this branch is a work in progress to add features and (unfortunately) breaking changes; here is a review of what changed compared to `ws4sqlite` "stable" + a migration path.

## Changes

### Breaking changes

- When running the app, the config files must be specified on the command line, the file paths cannot be used anymore (there). This is described in the "Migration" section below. The file path is in the config file.
  - The only exception is a "simple case" to serve a file path without any config. This can be done with the new `--quick-db` parameter.
- Hashed passwords in auth config must now be hashed with BCrypt, not SHA256.
- Plain text passwords are not permitted anymore, in auth config.

### Major features

- SQLite is embedded via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) and CGO. Should be way faster.
- Support for DuckDB (see below).
- Target platforms (because of CGO) are now 4 (`win/amd64`, `macos/arm64`, `linux/amd64`, `linux/arm64`).
  - For Docker, `linux/amd64` and `linux/arm64`.
- Docker images are now based on `distroless/static-debian12`.
- Docker images are now hosted on Github's Container Registry.

### Minor changes

- Fail fast if the request is empty, don't even attempt to authenticate.

## Migration

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

- For any hashed password (`HashedPassword = ...`) previously specified in an `auth` block, the hash must be BCrypt, not SHA256.
- For any plain text password (`Password = ...`), convert  in `HashedPassword`, also using BCrypt.

## Specific to DuckDB

- `noFail` is not supported.
- Accessing the same file from two different connections is not supported
- Placeholders for named parameters are `$VAL`, not `:VAL` as in SQLite.
- As DuckDB does not support read-only transactions, when `readOnly` is specified the requests won't be processed in a transaction.
- Duckdb exports backups in a folder. A backup is performed by exporting to a temp folder and zipping it. CSV format is used (`EXPORT DATABASE '...' (FORMAT CSV)`).
- At least for now, when instructed, a `VACUUM` is performed, not a `VACUUM ANALYZE`.

## Roadmap

1. Support mariadb/mysql
1. Support postgresql
1. ...
1. Profit!
