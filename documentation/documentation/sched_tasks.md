# 🔨 Scheduled Tasks

{% hint style="warning" %}
Since v0.14; this is an evolution of the former `maintenance` configuration, allowing for multiple schedulings. The old
form is deprecated but still supported for retrocompatibility; a warning will be displayed when using it, and an error
if both forms are used.
{% endhint %}

Going back to this snippet of [the configuration file](configuration-file.md):

```yaml
scheduledTasks:
  - schedule: "0 0 * * *"
    doVacuum: true
    doBackup: true
    backupTemplate: ~/first_%s.db
    numFiles: 3
    statements:
      - DELETE FROM myTable WHERE tstamp < CURRENT_TIMESTAMP - 3600
      - ... 
  - atStartup: true
    doVacuum: true
```

The `scheduledTasks` node represent the structure that tells ws4sqlite to provide execute tasks in a scheduled fashion.
This can be useful for maintenance, for example; each task can be:

- scheduled, with a cron-like syntax; and/or
- performed at startup.

The task itself can be comprised of one or more actions, i.e.:

- a VACUUM, to optimize and cleanup the internal structures;
- a backup, rotated as needed;
- a set of SQL statements, for example to cleanup temporary tables.

The last feature in particular is very powerful, in that it allows to perform statements at startup, or repeatedly; if
for example you need to generate a sort of "run id" for one particular run, the relevant SQL can be executed at each
startup of the server.

{% hint style="info" %}
If multiple actions are configured for a task, they are executed in the following order: first the VACUUM, then the 
backup, then the sql statements (in the order they're listed).
{% endhint %}

It's a list of objects. We'll now discuss the configurations of each node of the list.

#### `schedule`

_Line 2; string; it's mandatory to set either this or `atStartup` (as true)_

Cron-like string, standard 5-fields (no seconds). 
See [documentation](https://www.adminschoice.com/crontab-quick-reference) for more details.

{% hint style="warning" %}
It's always better to put double quotes (`"`) around the chron expression, as `*` is a special character for YAML.
{% endhint %}

#### `atStartup`

_Line 3, 10; boolean; it's mandatory to set either this (as true) or `schedule`_

If present and set to `true`, performs the task at engine startup.

#### `doVacuum`

_Line 4; boolean_

If present and set to `true`, performs a [`VACUUM`](https://www.sqlite.org/lang\_vacuum.html) on the database.

#### `doBackup`

_Line 5; boolean_

If present and set to `true`, performs a backup of the database according to the scheduling and the configurations.

The backup is created with the `VACUUM INTO...` command.

The following parameters tell ws4sqlite how to manage the backup(s).

{% hint style="info" %}
Depending on the configurations, it is possible that ws4sqlite creates more than one backup in the same minute; in this
case, the policy id _not_ to overwrite an existing file.
{% endhint %}

#### backupTemplate

_Line 6; string; mandatory if `doBackup` is `true`_

Filename (with path) of the backup files. It must contain a single `%s` that will be replaced with the minute of the 
backup, in `yyyyMMdd-HHmm` format. For example:

`../myFile_%s.db` will be generated as `../myFile_20220127-1330.db`

`~` is expanded to the user's home directory path.

#### numFiles

_Line 7; number; mandatory if `doBackup` is `TRUE`_

Indicates how many files to keep for each database. After the limit is reached, the files rotate, with the least 
recent files being deleted.

### statements

_Lines 8-10; list of strings_

A set of SQL statements (without parameters) to execute. 

{% hint style="warning" %}
The statements are not run in a transaction: if one fails, the next one will be executed, as with `initStatements`. On
the other hand, there is a mutex that ensures that the statements' block is not executed concurrently to a request.
{% endhint %}
