# ❓ Requests

A request is a JSON structure that is passed via a POST HTTP call to ws4sqlite, using the port specified when [running](running.md#port) the server application.

First and foremost, the database we connect to is specified in the URL of the POST call. It is something like this:

```bash
http://localhost:12321/db2
```

That `db2` part is the database ID, and must match the `id` of a database, defined in the [commandline](running.md#databases-and-config-companion-files).

Second, and very important, the request must contain the header:

```http
Content-type: application/json
```

This is a JSON that exemplifies all possible elements of a request.

```json5
{
    "resultFormat": "map",
    "credentials": {
        "user": "myUser1",
        "password": "myCoolPassword"
    },
    "transaction": [
        {
            "query": "SELECT * FROM TEMP"
        },
        {
            "query": "SELECT * FROM TEMP WHERE ID = ?",
            "values": [ 1 ] // Positional parameters
        },
        {
            "statement": "INSERT INTO TEMP (ID, VAL) VALUES (0, 'ZERO')"
        },
        {
            "noFail": true,
            "statement": "INSERT INTO TEMP (ID, VAL) VALUES (:id, :val)",
            "values": { "id": 1, "val": "a" } // Named parameters
        },
        {
            "statement": "#Q2",
            "valuesBatch": [
                { "id": 2, "val": "b" },
                { "id": 3, "val": "c" }
            ]
        }
    ]
}
```

Let's go through it.

#### Result Format

_Line 2; string; `map` or `list` (case insensitive); default `map`_

_Since 0.16_

This is the format for result sets, in the response. By default, it returns an (ordered) map with the results, but it can return a list of them, too.

[See response docs, here.](responses.md#list-format-for-resultsets)

#### Authentication Credentials

_Lines 3-6; object_

If [authentication](authentication.md) is enabled _in `INLINE` mode_, this object describes the credentials. See the [detailed docs](authentication.md#credentials-in-the-request-inline-mode).

#### List of Queries/Statements

_Line 7; list of objects; mandatory_

**Must be not empty**. The list of the queries or statements that will actually be performed on the database, with their own parameters.

They will be run in a transaction, and the transaction will be committed only if all the queries that are _not_ marked as `noFail` (see the [relevant section](errors.md)) do successfully complete.

#### SQL Statements to Execute

_Lines 9, 12, 16, 20; string; mandatory one of `query` or `statement`_

The actual SQL to execute.

Specifying it as `query` means that a result set is expected (typically, `SELECT` queries or queries with a `RETURNING` clause).

Specifying a `statement` will not return a result set, but a count of affected records.

#### Stored Query Reference

_Line 24; string; mandatory as the above_

A `query` or a `statement` (see above) can consist of a reference to a Stored Query. They are prepended by a `#`. An error will occour if the S.Q. with an ID equal to the part after the `#` was not defined for this database.

See the [relevant section](stored-statements.md).

#### Parameter Values for the Query/Statement

_Lines 13, 21; object_

If the query needs to be parametrized, named parameters can be defined in the statement using SQLite [syntax](https://www.sqlite.org/c3ref/bind\_blob.html) (e.g. `:id` or `@id`, or `?`), and the proper values for them must be specified here. You can specify values that do not match a parameter; they'll be ignored.

Since 0.16, there are two forms of parameters:

* **Positional**, as in Line 13; in the SQL you'll want to use the `?` form for placeholders, and then specify the parameters, in order, in a list;
* **Named**, as in line 21; in the SQL use `:name` or `@name`, and specify an object with their value. May help reduce clutter.

{% hint style="info" %}
Before v0.16, the parameters were always named.
{% endhint %}

{% hint style="warning" %}
What happens if some parameter values aren't defined in the `values` object, in its named form? If there are _less_ parameter values than expected, it will give an error. If they are correct in number, but some parameter names don't match, the missing parameters will be assigned a value of `null`.
{% endhint %}

#### Batch Parameter Values for a Statement

_Lines 25..28; list of objects_

Only for `statement`s, more than one set of parameter values can be specified; the statement will be applied to them in a batch (by _preparing_ the statement).

#### NoFail: Don't Fail when Errors Occour

_Line 29; Boolean_

When a query/statement fails, by default the whole transaction is rolled back and a response with a single error is returned (the first one for the whole transaction). Specifying this flag, the error will be reported for that statement, but the execution will continue and eventually be committed. See the [relevant page](errors.md) for more details.
