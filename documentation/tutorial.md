# 🏫 Tutorial

In this tutorial we'll run ws4sql for the first time, in order to serve a single database, and we'll run a couple of queries and statements against it.

We'll use Linux, but the information here is "portable" to MacOS and Windows as well.

> ℹ️ If ws4sql offers a relevant capability that doesn't fit in a tutorial, it will be explained in an info box like this, and a link to the relevant documentation will be provided.

Let's start!

### 🔧 Installation

The installation is simple, as ws4sql "is" just an executable file. [Download](https://github.com/proofrock/ws4sql/releases) or [build](building-and-testing.md) it (matching your OS and architecture), and put it somewhere on the filesystem.

### 🚀 First Run & Configuration

Let's now start the application:

```bash
./ws4sql --quick-db testDb.db
```

This tells ws4sql to serve a SQLite database, to be created because a file at the specified path doesn't exist, using default settings. It's now possible to access the database using its id, that is the filename minus the suffix (in this case, `testDb`).

> ℹ️ More than one database can be served from the same instance, and it's possible to create [in-memory databases](documentation/running.md#mem-db). Of course more options are possible: provide [authentication](documentation/authentication.md), open the file as [read only](documentation/configuration-file.md), [specify some queries/statements](documentation/stored-statements.md) on the server that can be referenced in requests, provide [initialization statements](documentation/configuration-file.md#initstatements) to apply when creating a database, and several more. This is done by creating a [companion YAML file](documentation/configuration-file.md) at the same path, called like the database but with a `.yaml` extension: `testDb.yaml` in our example.

When the app starts, something like this will be printed; it gives information about what is now being served, and how.

```
                __ __             __
 _      _______/ // / _________ _/ /
| | /| / / ___/ // /_/ ___/ __ `/ /
| |/ |/ (__  )__  __(__  ) /_/ / /
|__/|__/____/  /_/ /____/\__, /_/
                           /_/ v0.x.x
+ sqlite v3.49.1
+ duckdb v1.2.2
- Parsing config file: quick db setting
  + Serving database 'mydatabase'
  + File not present, it will be created
  + Using WAL
- Web Service listening on 0.0.0.0:12321
```

The service is now active and serving requests. Use `Ctrl-c` to exit, as usual.

> ℹ️ From the commandline, it's also possible to specify the [port](documentation/running.md#port) and the [host](documentation/running.md#bind-host) to bind to.

### 🔍 First Request

Let's now do something useful. Use a tool like [postman](https://www.postman.com) to submit a POST call to `http://localhost:12321/testDb`, with the following body:

```json
{
    "transaction": [
        {
            "statement": "CREATE TABLE TEST_TABLE (ID int primary key, VAL text)"
        }
    ]
}
```

> ⚠️ Ensure that the header `Content-Type` is set to `application/json`!

Let's see what is in the request:

* **URL:** in the connection URL, we specify the database ID to submit the request to, as defined in the config file;
* **Line 2**: specify the transaction operation list: an array of requests (queries or statements) to submit;
* **Line 4**: as the first (and only) request, specify a statement (i.e. a SQL command that doesn't return a resultset).

If all goes well, you should get a `200` response with the following body:

```json
{
    "results": [
        {
            "success": true,
            "rowsUpdated": 0
        }
    ]
}
```

You did it! 🚀 Going through the response:

* **Line 2**: the array of results has the same size of the corresponding `transaction`array in the request, and lists the results of each request, in turn;
* **Line 4**: specifies that the statement completed with success;
* **Line 5**: there were no updated rows (as reported by SQLite; this is a DDL command).

### 🤹 Multiple Requests

Let's say you want to run multiple SQLs in the same request. As you may suspect, this is just a matter of adding another item to the `transaction` array:

> ℹ️ As the name of the array implies, the queries/statements are run in a single transaction. It will be committed at the end of the call, and rolled back if an unmanaged error occours (see [the relevant chapter](tutorial.md#managing-errors)).

```json
{
    "transaction": [
        {
            "statement": "CREATE TABLE TEST_TABLE_2 (ID int primary key, VAL text)"
        },
        {
            "statement": "INSERT INTO TEST_TABLE_2 (ID, VAL) VALUES (1, 'hello')"
        }
    ]
}
```

The response now has 2 elements in the array:

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
        }
    ]
}
```

Please notice **Line 9**: `rowsUpdated` is 1, signaling that we affected one row with the `INSERT`.

### 🧯 Managing Errors

Let's send over the last request again (don't remove the file!). The table already exists, so the response will now fail with `500 Internal Server Error` , and with a body of:

```json
{
    "reqIdx": 0,
    "error": "Table TEST_TABLE_2 already exists",
}
```

* **Line 2**: the (0-based) index of the statement that failed in the `queries` array; an index of `-1` would tell us that it's a generic error, not tied to a particular statement;
* **Line 3**: the reason of the failure, as reported by the database driver.

This is a general failure. As statements in the same request are run in transaction, the whole transaction is rolled back.

**In SQLite** it's also possible to "allow" certain statements to fail. The transaction will be completed, and an error will be reported only on that statement. Send the following request, notice the `"noFail"` in the first statement:

```json
{
    "transaction": [
        {
            "noFail": true,
            "statement": "CREATE TABLE TEST_TABLE_2 (ID int primary key, VAL text)"
        },
        {
            "statement": "INSERT INTO TEST_TABLE_2 (ID, VAL) VALUES (1, 'hello')"
        }
    ]
}
```

The following result is produced, signaling that the first statement failed; the transaction is committed anyway, so the second statement is actually persisted:

```json
{
    "results": [
        {
            "success": false,
            "error": "Table TEST_TABLE_2 already exists"
        },
        {
            "success": true,
            "rowsUpdated": 1
        }
    ]
}
```

### 🎂 Queries With a Result (Set)

Up to now, we tested only statements, that don't return results other than the number of affected rows. Let's see how to run a _query_.

In the next example we will create a table, insert two rows in it, and read them.

> ⚠️ Please start from an empty database: stop ws4sql, remove the database file and start it again as described above.

Request:

```json
{
    "transaction": [
        {
            "statement": "CREATE TABLE TEST_TABLE (ID int primary key, VAL text)"
        },
        {
            "statement": "INSERT INTO TEST_TABLE (ID, VAL) VALUES (1, 'hello'), (2, 'world')"
        },
        {
            "query": "SELECT * FROM TEST_TABLE ORDER BY ID ASC"
        }
    ]
}
```

* **Line 10**: notice that the key now is `query`, to signal that it will generate a result set.

```json5
{
    "results": [
        // ...the first two results are omitted for clarity...
        {
            "success": true,
            "resultHeaders": [ "ID", "VAL" ],
            "resultSet": [
                { "ID": 1, "VAL": "hello" },
                { "ID": 2, "VAL": "world" }
            ]
        }
    ]
}
```

* **Lines 7..10**: we now have an array with two results, containing objects. Each object has several fields, with the key being the name of the database field and the value being... the value. The key/field name is as reported by the database (see **line 5**), so `*` works well.

### ♟️ Using Parameters

The last capability we'll cover is using parameters, either in a statement (e.g. an `INSERT`) or in a query. They are specified using named placeholders, like the following.

> ℹ️ The actual syntax for named parameters may be different for SQLite and DuckDB. `ws4sql` uses the "native" syntax of the database. Here SQLite's syntax is used.

```json
{
    "transaction": [
        {
            "statement": "INSERT INTO TEST_TABLE (ID, VAL) VALUES (:id, :val)",
            "values": { "id": 101, "val": "A hundred and 1" }
        },
        {
            "query": "SELECT * FROM TEST_TABLE WHERE ID = :id",
            "values": { "id": 101 }
        }
    ]
}
```

* **Line 4**: in the statement, we use named placeholders like `:id` ;
* **Line 5**: we specify the actual values with a `values` object, containing a map with the keys being the placeholders;
* Same with queries, as at **Line 9**.

> ℹ️ Using placeholders may seem more verbose than specifying the values in the SQL, but it is _always_ the preferrable solution, allowing for example to avoid nasty [SQL injection bugs](https://xkcd.com/327/).

> ℹ️ For statements, it is also possible to specify multiple sets of values ('batches'); the statement will be cached and replayed for each set of the list. See `valuesBatch` in the [docs](documentation/requests.md).

```json
{
    "results": [
        {
            "success": true,
            "rowsUpdated": 1
        },
        {
            "success": true,
            "resultHeaders": [ "ID", "VAL" ],
            "resultSet": [
                { "ID": 101, "VAL": "A hundred and 1" }
            ]
        }
    ]
}
```

As you can see, the response is the same, with only one result in the resultset (since the query selects by primary key).

### 🕯️ Conclusions

Thanks for reading so far, I hope you liked it! There are many more topics of interest, among which:

* Learn to protect your transactions with [authentication](documentation/authentication.md);
* Use a [reverse proxy](integrations/reverse-proxy.md) for HTTPS and additional security;
* Use [stored statements](documentation/stored-statements.md) to avoid passing SQL from the client;
* Perform scheduled activities, that is: sql statements, `VACUUM`s or backups;
* Configure [CORS](documentation/configuration-file.md#corsorigin) for more convenient access from a web page;
* ...and much more!

Have a nice day! ☀️
