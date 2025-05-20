# 🥇 Features

* Aligned to [**SQLite 3.49.1**](https://sqlite.org/releaselog/3\_49\_1.html);
* A [**single executable file**](https://germ.gitbook.io/ws4sqlite/documentation/installation) (written in Go);
* HTTP/JSON access, with [**client libraries**](https://germ.gitbook.io/ws4sqlite/client-libraries) for convenience;
* Directly call `ws4sqlite` on a database (as above), many options available using a YAML companion file;
* [**In-memory DBs**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#path) are supported;
* Serving of [**multiple databases**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file) in the same server instance;
* [**Batching**](https://germ.gitbook.io/ws4sqlite/documentation/requests#batch-parameter-values-for-a-statement) of multiple value sets for a single statement;
* [**Parameters**](../documentation/requests.md#batch-parameter-values-for-a-statement) may be passed to statements positionally (lists) or by name (maps);
* [**Results**](../documentation/responses.md#list-format-for-resultsets) of queries may be returned as key-value maps, or as values lists;
* All queries of a call are executed in a [**transaction**](https://germ.gitbook.io/ws4sqlite/documentation/requests);
* For each query/statement, specify if a failure should rollback the whole transaction, or the failure is [**limited**](https://germ.gitbook.io/ws4sqlite/documentation/errors#managed-errors) to that query;
* "[**Stored Statements**](https://germ.gitbook.io/ws4sqlite/documentation/stored-statements)": define SQL in the server, and call it from the client;
* [**CORS**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#corsorigin) mode, configurable per-db;
* [**Scheduled tasks**](https://germ.gitbook.io/ws4sqlite/documentation/sched\_tasks) (VACUUM, sql or backups), also configurable per-db;
* Provide [**initialization statements**](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#initstatements) to execute when a DB is created;
* [**WAL**](https://sqlite.org/wal.html) mode enabled by default, can be [disabled](https://germ.gitbook.io/ws4sqlite/documentation/configuration-file#disablewalmode);
* [**Embedded web server**](../documentation/web-server.md) to directly serve web pages that can access ws4sqlite without CORS;
* Quite fast!
* Compact codebase;
* Comprehensive test suite (`make test`);
* 11 os's/arch's directly supported;
* [**Docker images**](https://germ.gitbook.io/ws4sqlite/documentation/installation/docker), for amd64, arm and arm64.

### Security Features

* [**Authentication**](../security.md#authentication) can be configured
  * on the client, either using HTTP Basic Authentication or specifying the credentials in the request;
  * on the server, either by specifying credentials (also with hashed passwords) or providing a query to look them up in the db itself;
* A database can be opened in [**read-only mode**](../security.md#read-only-databases) (only queries will be allowed);
* It's possible to enforce using [**only stored statements**](../security.md#stored-statements-to-prevent-sql-injection), to avoid some forms of SQL injection and receiving SQL from the client altogether;
* [**CORS Allowed Origin**](../security.md#cors-allowed-origin) can be configured and enforced;
* It's possible to [**bind**](../security.md#binding-to-a-network-interface) to a network interface, to limit access.

Some design choices:

* Very thin layer over SQLite. Errors and type translation, for example, are those provided by the SQLite driver;
* Doesn't include HTTPS, as this can be done easily (and much more securely) with a [reverse proxy](../security.md#use-a-reverse-proxy-if-going-on-the-internet);
* Doesn't support SQLite extensions, to improve portability.
