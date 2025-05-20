# 📦 Stored Statements

As stated while discussing the [configuration file](configuration-file.md#storedqueries-lines-26-30-and-useonlystoredqueries-line-25), Stored Statements are a way to specify (some of) the statement/queries you will use in the server instead of sending them over from the client.

This can be done to shorten the requests from the client; a more important reason is for security: when coupled with the next parameter, this allows the server to actually limit the SQL tat is performed to a set of predefined, controlled values. If your db contains areas that are sensitive, and you don't want to expose them, this can be very effective. See also the [relevant section](../security.md#stored-queries-to-prevent-sql-injection).

#### Configuration

For any database, you can add a similar piece of YAML to specify one or more stored statements:

```yaml
storedStatements:
  - id: Q1
    sql: SELECT * FROM TABLE_1
  - id: Q2
    sql: SELECT * FROM TABLE_2 WHERE UPPER(cf) = UPPER(:cf)
```

As you can see, each stored statement has an ID that you can use to refer to it from a request.

#### Usage in requests

Simply specify the ID, prepended by a `#` sign, instead of any SQL:

```json
{
    "transaction": [
        {
            "query": "#Q1"
        }
    ]
}
```

Of course, despite the term, it's not limited to statements, but also `query`s works the same. "Statement" is just used in a generic sense here.

#### Limiting the server to executing stored statements

Specify the relevant configuration (line 2):

```yaml
[...]
useOnlyStoredStatements: true
storedStatements:
[...]
```

This way, the server will not accept any SQL from the client, but will only allow Stored Statements.

A possible use case is this: you have a database with a table that lists some sensitive data for a key.

```sql
CREATE TABLE MY_SENSITIVE_TABLE (
  MY_KEY     TEXT NOT NULL PRIMARY KEY,
  MY_SECRET  TEXT NOT NULL
)
```

You want to avoid that the whole table is extracted, but still keep it accessible for queries for a single key.

Just define a query, and restrict the server to it:

```yaml
[...]
useOnlyStoredStatements: true
storedStatements:
  - id: Q1
    sql: SELECT MY SECRET FROM MY_SENSITIVE_TABLE WHERE MY_KEY = :key
```

From the client, anyone will only be allowed to perform a lookup by key, so they'll need to know the key in order to access the data. No other statements are allowed, and no SQL can be passed from the client.
