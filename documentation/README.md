---
description: Query sqlite or duckdb via http - and remote clients too!
---

## 🌱 Introduction & Credits

**ws4sql** is a server-side application that, applied to one or more sqlite or duckdb files, allows to perform SQL queries and statements on them via REST (or better, JSON over HTTP).

> ℹ️ `ws4sql` was formerly called `ws4sqlite` because, well, it supported only sqlite. It's in the roadmap to add other databases, too, like postgresql or mariadb.

> ℹ️ [This document](ROAD_TO_WS4SQL.md) outlines all the changes between `ws4sqlite` and `ws4sql`, with hints for a successful migration.

Possible use cases are the ones where remote access to a sqlite db is useful/needed, for example a data layer for a remote application, possibly serverless or even called from a web page (_after security considerations_ of course).

Client libraries are available, that will abstract the "raw" JSON-based communication. See [here](https://github.com/proofrock/ws4sqlite-client-jvm) for Java/JVM, [here](https://github.com/proofrock/ws4sqlite-client-go) for Go(lang); others will follow.

As a quick example for SQLite: after launching

```bash
./ws4sql --quick-db mydatabase.db
```

It's possible to make a POST call to `http://localhost:12321/mydatabase`, e.g. with the following body:

```json5
// Set Content-type: application/json
{
    "transaction": [
        {
            "statement": "CREATE TABLE TEST_TABLE (ID INTEGER PRIMARY KEY, VAL TEXT)"
        },
        {
            "statement": "INSERT INTO TEST_TABLE (ID, VAL) VALUES (:id, :val)",
            "values": { "id": 1, "val": "hello world" }
        },
        {
            "query": "SELECT * FROM TEST_TABLE"
        }
    ]
}
```

Obtaining an answer of:

```json
{
    "results": [
        {
            "success": true,
            "rowsUpdated": 0
        },
        {
            "success": true,
            "rowsUpdated": 1
        },
        {
            "success": true,
            "resultHeaders": [ "ID", "VAL" ],
            "resultSet": [
                { "ID": 1, "VAL": "hello world" }
            ]
        }
    ]
}
```

### Credits

Many thanks and all the credits to these awesome projects:

* [lnquy's cron](https://github.com/lnquy/cron) (MIT License);
* [robfig's cron](https://github.com/robfig/cron) (MIT License);
* [gofiber's fiber](https://github.com/robfig/cron) (MIT License);
* [marcboeker's go-duckdb](https://github.com/marcboeker/go-duckdb) (MIT License);
* [mitchellh's go-homedir](https://github.com/mitchellh/go-homedir) (MIT License);
* [mattn's go-sqlite3](https://github.com/mattn/go-sqlite3) (MIT License);
* [wI2L's jettison](https://github.com/wI2L/jettison) (MIT License)
* [iancoleman's orderedmap](https://github.com/iancoleman/orderedmap) (MIT License);
* and of course, [Google Go](https://go.dev).

Kindly supported by [JetBrains for Open Source development](https://jb.gg/OpenSourceSupport)
