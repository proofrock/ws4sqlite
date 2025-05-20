# 🔦 Cheat Sheet

## Commandline

```bash
ws4sql 
    --bind-host 0.0.0.0 \        # Optional
    --port 12321 \               # Optional
    --db ~/file1.db \            # File-based db, cfg at ~/file1.yaml
    --mem-db mem1:~/mem1.yaml \  # Memory-based db, with cfg
    --mem-db mem2 \              # Memory-based db, with default cfg
    --serve-dir myDir            # Serve static resources from a filesystem directory
```

## Configuration file

```yaml
# All the first-level elements are optional (auth, disableWALMODE, ...)
auth:
  mode: HTTP                      # INLINE or HTTP
  customErrorCode: 499            # If 401 is not optimal
  # Specify one of byQuery or byCredentials
  byQuery: SELECT 1 FROM AUTH WHERE USER = :user AND PASSWORD = :password
  byCredentials:                  # The query must have :user and :password
    - user: myUser1
      password: myCoolPassword
    - user: myUser2
      hashedPassword: b133...     # SHA-256 of the password in HEX form
disableWALMode: true
readOnly: true
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
  - atStartup: true               # This (as true) or schedule must be present
    doVacuum: true
corsOrigin: https://myownsite.com # Access-Control-Allow-Origin
useOnlyStoredStatements: true     
storedStatements:
  - id: Q1
    sql: SELECT * FROM TEMP 
  - id: Q2
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

```json
{
    "credentials": {                      // Necessary if and only if auth.mode == INLINE
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
        {
            "noFail": true,
            "statement": "INSERT INTO TEMP (ID, VAL) VALUES (:id, :val)",
            "values": { "id": 1, "val": "a" }
        },
        {
            "statement": "#Q2",            // '#' + the ID of the Stored Statement
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

```json
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
            "success": false,
            "error": "UNIQUE constraint failed: TEMP.ID"
        },
        {
            "success": true,
            "rowsUpdatedBatch": [ 1, 1 ]
        }
    ]
}
```
