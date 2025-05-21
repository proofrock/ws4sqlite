> **This page is not yet updated**

# 🏃 Running

Running ws4sql can be done via the commandline, and it's possible to specify its behaviour via commandline parameters.

### Databases, `id`'s and Config (Companion) Files

ws4sql behaviour when serving databases is configured with a mix of commandline parameters and naming conventions. You specify directly on the commandline the databases to serve, either specifying the path of the file (for file-based databases) or the id of the in-memory database. Then, for file paths, ws4sql will look for a ["companion" configuration file](configuration-file.md) in YAML format that have the same base name of the database filename, but with `.yaml` extension. When present, it is used to load the serving parameters for that database.

For in-memory dbs, you indicate an id to be used for that database, followed - if needed - by the path of the companion file for the in-memory database, separated by a colon (`:`).

In both cases, if the companion file is not present, default values will be used.

For example, if you run ws4sql as such:

```bash
ws4sql --db ~/file1.db --mem-db mem1:~/mem1.yaml --mem-db mem2
```

It will:

* serve a db from `~/file1.db`, creating it if absent, with an id of `file1`;
* look for `~/file1.yaml`, and - if present - load from there the configuration for this db;
* serve a db from memory, with an id of `mem1`;
* load its configuration from `~/mem1.yaml`;
* serve a db from memory, with an id of `mem2`, and default configuration.

{% hint style="info" %}
Since version 0.11.3, any extension can be used database file. Before that version, only `.db` could be used.
{% endhint %}

### Commandline Parameters

This is a complete commandline for ws4sql:

```bash
ws4sql --bind-host 0.0.0.0 --port 12321 --db FILE_DB --mem-db MEM_DB_ID[:MEM_DB_CFG_FILE]
```

Of course, the usual `--help` and `--version` are supported. Let's discuss the other commandline parameters one by one.

#### `--bind-host`

Host to bind to. Defaults to `0.0.0.0`, meaning to accept connections from any local network interface. Different values are possible to restrict connections to a particular subnet.

#### `--port`

Port to use for incoming network communication. Defaults to `12321`.

#### `--db`

Can be repeated.

Specifies one or more file paths to load and serve as SQLite db's. It will use the base name (without the `.db` suffix)
as the id of the database, to use in the URL of the [request](requests.md) , and will look for a configuration/companion
file in the same path, named `<id>.yaml`.

It is also possible to specify a companion file at a different path, by specifying it after a colon (`:`).  Example: 
`--db myFile.db:/another/path/myFileConfig.yaml`.

#### `--mem-db`

Can be repeated.

Specifies one or more id for in-memory databases. Optionally, it's possible to specify also the path of the configuration
file, after a colon (`:`).

See the example above for a clearer explanation.

#### `--serve-dir`

Specifies a directory to serve via the internal web server. See the [relevant docs page](web-server.md).

### Output

ws4sql will parse the commandline and the (eventual) [config files](configuration-file.md), attempt to open and connect to the various databases, creating their respective files as needed. Then it will output a summary of all the configurations, like this:

```
ws4sql x.y.z
- Serving directory '/my/web/contents'
- Serving database 'db1' from /data/db1.db?_journal=WAL
  + Parsed companion config file
  + Using WAL
  + Authentication enabled, with 1 credentials
  + Maintenance scheduled at 12:01 am
  + CORS Origin set to https://db1.myserver.it
- Serving database 'db2' from /data/db2.db?mode=ro&immutable=1&_query_only=1&_journal=WAL
  + Parsed companion config file
  + Using WAL
  + Read only
  + Strictly using only stored statements
  + With 2 stored statements
  + CORS Origin set to https://noi.sbertilla.it
- Web Service listening on 0.0.0.0:12321
```
