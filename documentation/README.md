---
description: Query sqlite or duckdb via http - and remote clients too!
---

## 🌱 Introduction & Credits

**ws4sql** is a server-side application that, applied to one or more SQLite files, allows to perform SQL queries and statements on them via REST (or better, JSON over HTTP).

Possible use cases are the ones where remote access to a sqlite db is useful/needed, for example a data layer for a remote application, possibly serverless or even called from a web page (_after security considerations_ of course).

Client libraries are available, that will abstract the "raw" JSON-based communication. See [here](https://github.com/proofrock/ws4sqlite-client-jvm) for Java/JVM, [here](https://github.com/proofrock/ws4sqlite-client-go) for Go(lang); others will follow.

As a quick example, after launching

```bash
ws4sql --db mydatabase.db
```

It's possible to make a POST call to `http://localhost:12321/mydatabase`, e.g. with the following body:

```json5
// Set Content-type: application/json
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

Obtaining an answer of:

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

#### Credits

Many thanks and all the credits to these awesome projects:

* [lnquy's cron](https://github.com/lnquy/cron) (MIT License);
* [robfig's cron](https://github.com/robfig/cron) (MIT License);
* [gofiber's fiber](https://github.com/gofiber/fiber) (MIT License);
* [klauspost's compress](https://github.com/klauspost/compress) (3-Clause BSD license);
* [mitchellh's go-homedir](https://github.com/mitchellh/go-homedir) (MIT License);
* [modernc.org's sqlite](https://gitlab.com/cznic/sqlite) (3-Clause BSD License);
* [wI2L's jettison](https://github.com/wI2L/jettison) (MIT License)
* and of course, [Google Go](https://go.dev).

Kindly supported by [JetBrains for Open Source development](https://jb.gg/OpenSourceSupport)
