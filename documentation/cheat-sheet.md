# 🔦 Cheat Sheet

## Commandline

```bash
ws4sql 
    --bind-host 0.0.0.0 \        # Optional
    --port 12321 \               # Optional
    --db ~/file1.yaml \          # Configuration file/pointer for a db
    --quick-db myFile.db \       # Single-file SQLite db, without configuration
    --serve-dir myDir            # Serve static resources from a filesystem directory
```

## Configuration file

```yaml
database:
  type: SQLITE          # SQLITE or DUCKDB. If omitted, defaults to SQLITE      
  inMemory: false       # The db is a memory one? If omitted, defaults to false
  path: ".../test.db"   # The db file path.
  id: test              # If omitted and !inMemory, calculates it from the file name
  disableWALMode: false # If type = SQLITE, disables WAL mode
  readOnly: false       # If true, doesn't allow modifications to db
# All the following first-level elements are optional (auth, disableWALMODE, ...)
auth:
  mode: HTTP                      # INLINE or HTTP
  customErrorCode: 499            # HTTP Code when auth fails; can be customized if the default (401) is not optimal
  # Specify one of byQuery or byCredentials
  # The query must have :user and :password for SQLite or $user/$password for DuckDB
  byQuery: SELECT 1 FROM AUTH WHERE USER = :user AND PASSWORD = :password
  # Credentials can be multiple, with different <user>, and the password may be BCrypt or cleartext
  byCredentials:                  
    - user: myUser1
      password: myCoolPassword
    - user: myUser2
      hashedPassword: b133...     # BCrypt hash of the password
schedTasks:                       # Multiple tasks are possible
  - schedule: "0 0 * * *"         # Cron format without seconds (m h d m wd)
    atStartup: false              # This (as true) or schedule must be present
    doVacuum: true
    doBackup: true
    backupTemplate: ~/temp_%s.db  # A placeholder %s must be present, it will be replaced with yyyyMMdd_HHmm
    numFiles: 3                   # Backup files to keep 
    statements:                   # SQL Statements to execute at every scheduled run
      - DELETE FROM myTable WHERE tstamp < CURRENT_TIMESTAMP - 3600
      - ...
  - atStartup: true               # Either this must be true, or a schedule must be present
    doVacuum: true
corsOrigin: https://myownsite.com # Access-Control-Allow-Origin
useOnlyStoredStatements: true     # Doesn't allow free-text SQL in the request, but only stored statements
storedStatements:
  - id: Q1                        # Refer as #Q1 in requests
    sql: SELECT * FROM TEMP 
  - id: Q2                        # Refer as #Q2 in requests
    sql: INSERT INTO TEMP VALUES (:id, :val)
initStatements:                   # These statements will be executed when a db is created
  - CREATE TABLE AUTH (USER TEXT PRIMARY KEY, PASSWORD TEXT)
  - INSERT INTO AUTH VALUES ('myUser1', 'myCoolPassword')
  - CREATE TABLE TEMP (ID INT PRIMARY KEY, VAL TEXT)
  - INSERT INTO TEMP (ID, VAL) VALUES (1, 'ONE'), (4, 'FOUR')
```

## Request

### URL

```
http://localhost:12321/<dbId>
```

### Headers

```
Content-Type: application/json
// + Basic Auth, if auth.mode == HTTP
```

### Body

```json5
{
    "resultFormat": "RESULTSET" // "LIST" gives the results as arrays. For any other value, ResultSets are returned, as below
    "credentials": {            // Necessary if and only if auth.mode == INLINE
        "user": "myUser1",
        "password": "myCoolPassword"
    },
    "transaction": [
        {
            "query": "SELECT * FROM TEMP"
        },
        {
            "query": "SELECT * FROM TEMP WHERE ID = :id",
            "values": { "id": 1 }
        },
        {
            "statement": "INSERT INTO TEMP (ID, VAL) VALUES (0, 'ZERO')"
        },
        // Named parameters
        {
            "noFail": true,     // Only for SQLite
            "statement": "INSERT INTO TEMP (ID, VAL) VALUES (:id, :val)", // For SQLite; '($id, $val)' for DuckDB
            "values": { "id": 1, "val": "a" }
        },
        // Positional parameters
        {
            "statement": "INSERT INTO TEMP (ID, VAL) VALUES (?, ?)",
            "values": [ 2, "b" ]
        },
        {
            "statement": "#Q2", // '#' + the ID of the Stored Statement
            "valuesBatch": [
                { "id": 2, "val": "b" },
                { "id": 3, "val": "c" }
            ]
        }
    ]
}
```

## Response

### General Error (`400`, `401`, `404`, `500`)

```json
{
    "reqIdx": 1,     // 0-based index of the failed subrequest; -1 for general
    "message": "near \"SELECTS\": syntax error"
}
```

### Success (`200`)

```json5
{
    "results": [
        {
            "success": true,
            "resultSet": [
                { "ID": 1, "VAL": "ONE" },
                { "ID": 4, "VAL": "FOUR" }
            ]
        },
        {
            "success": true,
            "resultSet": [
                { "ID": 1, "VAL": "ONE" }
            ]
        },
        {
            "success": true,
            "rowsUpdated": 1
        },
        {
            "success": false, // because "noFail" = true, it doesn't fail globally
            "error": "UNIQUE constraint failed: TEMP.ID"
        },
        {
            "success": true,
            "rowsUpdated": 1
        },
        {
            "success": true,
            "rowsUpdatedBatch": [ 1, 1 ]
        }
    ]
}
```

If `resultFormat` is `LIST` in the request, the result will be in the form:

```json5
{
    "results": [
        {
            "success": true,
            "resultSetList": [
                [ 1, "ONE" ],
                [ 4, "FOUR" ]
            ]
        },
        // ...
    ]
}
```
