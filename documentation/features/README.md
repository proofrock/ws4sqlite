# 🥇 Features

- A [**single executable file**](documentation/installation) (written in Go);
- Support for **SQLite** or **DuckDB** (more in the future);
- Aligned to [**SQLite 3.46.1**](https://sqlite.org/releaselog/3_46_1.html) and [**DuckDB 1.1.3**](https://github.com/duckdb/duckdb/releases/tag/v1.1.3);
- HTTP/JSON access, with [**client libraries**](client-libraries) for convenience;
- Directly call `ws4sql` on a database (as in the [README](README) example), many options available using a YAML companion file;
- [**In-memory DBs**] are supported (documentation/configuration-file#path);
- Serving of [**multiple databases**](documentation/configuration-file) in the same server instance;
- [**Batching**](documentation/requests#batch-parameter-values-for-a-statement) of multiple value sets for a single statement;
- **Parameters** may be passed to statements positionally (lists) or by name (maps);
- **Results** of queries may be returned as key-value maps, or as values lists;
- All queries of a call are executed in a [**transaction**](documentation/requests);
- For each query/statement, specify if a failure should rollback the whole transaction, or the failure is [**limited**](documentation/errors#managed-errors) to that query _[SQLite only]_;
- "[**Stored Statements**](documentation/stored-statements)": define SQL in the server, and call it from the client;
- [**CORS**](documentation/configuration-file#corsorigin) mode, configurable per-db;
- [**Scheduled tasks**](documentation/sched_tasks), cron-like and/or at startup, also configurable per-db;
- Scheduled tasks can be: backup (with rotation), vacuum and/or a set of SQL statements;
- Provide [**initialization statements**](documentation/configuration-file#initstatements) to execute when a DB is created;
- [**WAL**](https://sqlite.org/wal.html) mode enabled by default, can be [disabled](documentation/configuration-file#disablewalmode) _[SQLite only]_;
- [**Embedded web server**](documentation/web-server) to directly serve web pages that can access ws4sqlite without CORS;- [Quite fast](features/performances.md)!
- Comprehensive test suite (`make test`);
- [**Docker images**](documentation/installation/docker), for both amd64 and aarch64.

### Security Features

* [**Authentication**](documentation/security.md#authentication) can be configured
  * on the client, either using HTTP Basic Authentication or specifying the credentials in the request;
  * on the server, either by specifying credentials (also with BCrypt hashed passwords) or providing a query to look them up in the db itself;
  * customizable `Not Authorized` error code (if 401 is not optimal)
* A database can be opened in [**read-only mode**](documentation/security.md#read-only-databases) (only queries will be allowed);
* It's possible to enforce using [**only stored statements**](documentation/security.md#stored-statements-to-prevent-sql-injection), to avoid some forms of SQL injection and receiving SQL from the client altogether;
* [**CORS Allowed Origin**](documentation/security.md#cors-allowed-origin) can be configured and enforced;
* It's possible to [**bind**](documentation/security.md#binding-to-a-network-interface) to a network interface, to limit access.

Some design choices:

* Very thin layer over SQLite/DuckDB. Errors and type translation, for example, are those provided by the respective driver;
* Doesn't include HTTPS, as this can be done easily (and much more securely) with a [reverse proxy](documentation/security.md#use-a-reverse-proxy-if-going-on-the-internet);
* Doesn't support SQLite extensions, to improve portability; selected extensions for DuckDB are available.
