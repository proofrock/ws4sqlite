# 🌱 ws4sql

> I recently started a [discussion](https://github.com/proofrock/ws4sql/discussions/44) over the future direction for this project. Take a look, and chip in if you want!

**`ws4sql`** is a server application that, applied to one or more database files, allows to perform SQL queries and statements on them via REST (or better, JSON over HTTP). It supports SQLite and DuckDB.

Possible use cases are the ones where remote access to an embedded db is useful/needed, for example a data layer for a remote application, possibly serverless or even called from a web page (*after security considerations* of course).

Client libraries are available, that will abstract the "raw" JSON-based communication. See
[here](https://github.com/proofrock/ws4sqlite-client-jvm) for Java/JVM, [here](https://github.com/proofrock/ws4sqlite-client-go) for Go(lang); others will follow.

As a quick example, after launching

```bash
ws4sql --db mydatabase.db
```

It's possible to make a POST call to `http://localhost:12321/mydatabase`, e.g. with the following body:

```json5
// Set Content-type: application/json
{
    "resultFormat": "map", // "map" or "list"; if omitted, "map"
    "transaction": [
        {
            "statement": "INSERT INTO TEST_TABLE (ID, VAL, VAL2) VALUES (:id, :val, :val2)",
            "values": { "id": 1, "val": "hello", "val2": null }
        },
        {
            "query": "SELECT * FROM TEST_TABLE"
        }
    ]
}
```

Obtaining an answer of

```json
{
    "results": [
        {
            "success": true,
            "rowsUpdated": 1
        },
        {
            "success": true,
            "resultSet": [
                { "ID": 1, "VAL": "hello", "VAL2": null }
            ]
        }
    ]
}
```

# Features

[Docs](https://germ.gitbook.io/ws4sql/), a [Tutorial](https://germ.gitbook.io/ws4sql/tutorial), a [Discord](https://discord.gg/nBCcq2VQPu).

- Aligned to [**SQLite 3.46.1**](https://sqlite.org/releaselog/3_46_1.html) and [**DuckDB 1.1.3**](https://github.com/duckdb/duckdb/releases/tag/v1.1.3);
- A [**single executable file**](https://germ.gitbook.io/ws4sql/documentation/installation) (written in Go);
- HTTP/JSON access, with [**client libraries**](https://germ.gitbook.io/ws4sql/client-libraries) for convenience;
- Directly call `ws4sql` on a database (as above), many options available using a YAML companion file;
- [**In-memory DBs**](https://germ.gitbook.io/ws4sql/documentation/configuration-file#path)  are supported;
- Serving of [**multiple databases**](https://germ.gitbook.io/ws4sql/documentation/configuration-file) in the same server instance;
- [**Batching**](https://germ.gitbook.io/ws4sql/documentation/requests#batch-parameter-values-for-a-statement) of multiple value sets for a single statement;
- [**Parameters**](https://germ.gitbook.io/ws4sql/documentation/requests#parameter-values-for-the-query-statement) may be passed to statements positionally (lists) or by name (maps);
- [**Results**](https://germ.gitbook.io/ws4sql/documentation/responses#list-format-for-resultsets) of queries may be returned as key-value maps, or as values lists;
- All queries of a call are executed in a [**transaction**](https://germ.gitbook.io/ws4sql/documentation/requests);
- For each query/statement, specify if a failure should rollback the whole transaction, or the failure is [**limited**](https://germ.gitbook.io/ws4sql/documentation/errors#managed-errors) to that query;
- "[**Stored Statements**](https://germ.gitbook.io/ws4sql/documentation/stored-statements)": define SQL in the server, and call it from the client;
- [**CORS**](https://germ.gitbook.io/ws4sql/documentation/configuration-file#corsorigin) mode, configurable per-db;
- [**Scheduled tasks**](https://germ.gitbook.io/ws4sql/documentation/sched_tasks), cron-like and/or at startup, also configurable per-db;
- Scheduled tasks can be: backup (with rotation), vacuum and/or a set of SQL statements;
- Provide [**initialization statements**](https://germ.gitbook.io/ws4sql/documentation/configuration-file#initstatements) to execute when a DB is created;
- (for SQLite) [**WAL**](https://sqlite.org/wal.html) mode enabled by default, can be [disabled](https://germ.gitbook.io/ws4sql/documentation/configuration-file#disablewalmode);
- [**Quite fast**](features/performances.md)!
- [**Embedded web server**](https://germ.gitbook.io/ws4sql/documentation/web-server) to directly serve web pages that can access ws4sql without CORS;
- Compact codebase;
- Comprehensive test suite (`make test`);
- 6 os's/arch's directly supported;
- [**Docker images**](https://germ.gitbook.io/ws4sql/documentation/installation/docker), for amd64, arm and arm64.

# Security Features

- [**Authentication**](documentation/security.md#authentication) can be configured
  - on the client, either using HTTP Basic Authentication or specifying the credentials in the request;
  - on the server, either by specifying credentials (also with hashed passwords) or providing a query to look them up in the db itself;
  - customizable `Not Authorized` error code (if 401 is not optimal)
- A database can be opened in [**read-only mode**](documentation/security.md#read-only-databases) (only queries will be allowed);
- It's possible to enforce using [**only stored statements**](documentation/security.md#stored-statements-to-prevent-sql-injection), to avoid some forms of SQL injection and receiving SQL from the client altogether;
- [**CORS Allowed Origin**](documentation/security.md#cors-allowed-origin) can be configured and enforced;
- It's possible to [**bind**](documentation/security.md#binding-to-a-network-interface) to a network interface, to limit access.

# Design Choices

Some design choices:

- Very thin layer over the database. Errors and type translation, for example, are those provided by the db driver;
- Doesn't include HTTPS, as this can be done easily (and much more securely) with a [reverse proxy](documentation/security.md#use-a-reverse-proxy-if-going-on-the-internet);
- Doesn't support SQLite/DuckDB extensions, to improve portability.

# Contacts and Support

Let's meet on [Discord](https://discord.gg/nBCcq2VQPu)!

# Credits

Many thanks and all the credits to these awesome projects:

- [lnquy's cron](https://github.com/lnquy/cron) (MIT License);
- [robfig's cron](https://github.com/robfig/cron) (MIT License);
- [gofiber's fiber](https://github.com/gofiber/fiber) (MIT License);
- [klauspost's compress](https://github.com/klauspost/compress) (3-Clause BSD license);
- [mitchellh's go-homedir](https://github.com/mitchellh/go-homedir) (MIT License);
- [mattn's go-sqlite3](https://github.com/mattn/go-sqlite3) (MIT License);
- [modernc.org's go-duckdb](https://github.com/marcboeker/go-duckdb) (MIT License);
- [wI2L's jettison](https://github.com/wI2L/jettison) (MIT License)
- and of course, [Google Go](https://go.dev).

Kindly supported by [JetBrains for Open Source development](https://jb.gg/OpenSourceSupport)

# GPG Signing

Import my [key](https://public.germanorizzo.it/4BD993A579025E4932C0110DC368E8BA7D4453F6.gpgkey):

```bash
curl -sSL https://public.germanorizzo.it/4BD993A579025E4932C0110DC368E8BA7D4453F6.gpgkey | gpg --import -
```
