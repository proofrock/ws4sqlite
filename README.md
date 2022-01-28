# 🌱 ws4sqlite

**ws4sqlite** is a server-side application that, applied to one or more SQLite files, gives the possibility to perform SQL queries and statements on them via REST (or better, JSON over HTTP).

Possible use cases are the ones where remote access to a sqlite db is useful/needed, for example a data layer for a remote application, possibly serverless or even called from a web page (*after security considerations* of course).

As a quick example, after launching it on a file `mydatabase.db`, it's possible to make a POST call to `http://localhost:12321/mydatabase`, with the following body:

```json
{
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

[Docs](https://germ.gitbook.io/ws4sqlite/) and a [Tutorial](https://germ.gitbook.io/ws4sqlite/tutorial).

- A [**single executable file**](https://germ.gitbook.io/ws4sqlite/documentation/installation) (it's written in Go);
- Can load [**several databases**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file) at once, for convenience;
- [**Batching**](https://germ.gitbook.io/ws4sqlite/documentation/requests#batch-parameter-values-for-a-statement) of multiple values set for a single statement;
- All queries of a call are executed in a [**transaction**](https://germ.gitbook.io/ws4sqlite/documentation/requests);
- For each query/statement, specify if a failure should rollback the whole transaction, or the failure is [**limited**](https://germ.gitbook.io/ws4sqlite/documentation/errors#managed-errors) to that query;
- "[**Stored Statements**](https://germ.gitbook.io/ws4sqlite/documentation/stored-statements)": define SQL in the server, and call it with a key from the client;
- [**CORS**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#corsorigin) mode, configurable per-db;
- [**Maintenance**](https://germ.gitbook.io/ws4sqlite/documentation/maintenance) scheduling (VACUUM and backups), also configurable per-db;
- Builtin [**encryption**](https://germ.gitbook.io/ws4sqlite/documentation/encryption) of fields, given a symmetric key;
- Supports [**in-memory DBs**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#path);
- Allows to provide [**initialization statements**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#initstatements) to execute when a DB is created;
- [**WAL**](https://sqlite.org/wal.html) mode enabled by default, can be [disabled](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#disablewalmode);
- Very fast (benchmarks coming up!);
- Compact codebase (< 800 lines of code);
- Comprehensive test suite (`cd src; go test -v`);
- [**Docker images**](https://germ.gitbook.io/ws4sqlite/documentation/installation/docker) are available, both for amd64 and arm32.

# Security Features

- A database can be opened in [**read-only mode**](https://germ.gitbook.io/ws4sqlite/security#read-only-databases) (only queries will be allowed);
- [**Authentication**](https://germ.gitbook.io/ws4sqlite/security#authentication) can be configured
  - on the client, either using HTTP Basic Authentication or specifying the credentials in the request;
  - on the server, either by specifying credentials (also with hashed passwords) or providing a query to look them up in the db itself;
- It's possible to enforce using [**only stored statements**](https://germ.gitbook.io/ws4sqlite/security#stored-statements-to-prevent-sql-injection), to avoid some forms of SQL injection and receiving SQL from the client altogether;
- [**CORS Allowed Origin**](https://germ.gitbook.io/ws4sqlite/security#cors-allowed-origin) can be configured and enforced;
- It's possible to [**bind**](https://germ.gitbook.io/ws4sqlite/security#binding-to-a-network-interface) to a network interface, to limit access.

# Design Choices

Some deliberate choices have been made:

- Very thin layer over SQLite. Errors and type translation, for example, are those provided by the SQLite driver;
- Doesn't include HTTPS, as this can be done easily (and much more securely) with a [reverse proxy](https://germ.gitbook.io/ws4sqlite/security#use-a-reverse-proxy-if-going-on-the-internet);
- Same for HTTP compression or HTTP2: if you need to expose the service on the internet, you'll need a reverse proxy anyway;
- Doesn't support SQLite extensions, to improve portability.

# Credits

Many thanks and all the credits to these awesome projects:

- [mattn's go-sqlite3](https://github.com/mattn/go-sqlite3) (MIT License);
- [gofiber](https://github.com/gofiber/fiber) (MIT License);
- [robfig's cron](https://github.com/robfig/cron) (MIT License);
- [lnquy's cron](https://github.com/lnquy/cron) (MIT License);
- [mitchellh's go-homedir](https://github.com/mitchellh/go-homedir) (MIT License);
- [DataDog's zstd](https://github.com/DataDog/zstd) (Simplified BSD license);
- and of course, [Google Go](https://go.dev), [VS Code](https://code.visualstudio.com) and [CodeServer](https://github.com/coder/code-server)!
