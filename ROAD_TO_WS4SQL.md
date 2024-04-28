The next version of `ws4sqlite` will be called `ws4sql`, because it will support more RDBMs than sqlite.

The version in this branch is a work in progress to slowly add features and (unfortunately) breaking changes; here is a review of what changed compared to `ws4sqlite` "stable" + a migration path.

# Changes

- SQLite is embedded via [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) and CGO. Should be way faster.
- Target platforms (because of CGO) are now 6 (`win/amd64`, `macos/amd64`, `macos/arm64`, `linux/amd64`, `linux/arm64`, `linux/arm6`)

# Migration

- Nothing needed for now :-)

# Roadmap

1. Make it mandatory to use conf files for databases
1. Support duckdb (and iron out all the incompatibilities)
1. Support mariadb/mysql
1. Support postgresql
1. ...
1. Profit!