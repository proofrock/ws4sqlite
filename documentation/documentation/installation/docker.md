# 🐳 Docker

ws4sql provides a standard Docker image, based on distroless/static-debian12. It's a multiarch image for amd64 and aarch64/arm64.

Here are the relevant configurations:

|              |         |                           |
| ------------ | ------- | ------------------------- |
| Exposed port | 12321   | Fixed; remap it with `-p` |
| Config dir   | `/data` | Fixed; remap it with `-v` |
| User to run as | `--user <user_id>[:<group_id>]` | Docker standard switch; _**do use it**_ |
| Timezone | `-e TZ=xxx/yyy` | Docker standard env var |

#### Example

```bash
docker run -d \
 --restart=unless-stopped \
 --name=ws4sql \
 -p 8080:12321 \
 -v /mnt/DockerHome/myDir:/data \
 --user 1000:1001 \
 -e TZ=Europe/Rome \
 ghcr.io/proofrock/ws4sql:latest \
 --db /data/testDb.db
```

This command will install and run ws4sql, configuring it to:

* Use port 8080 (Line 4);
* Map a local dir to path `/data` in the docker environment (Line 5);
* Starts ws4sql as user 1000, group 1001 (Line 6);
* Sets the time zone (Line 7);
* Use free cli arguments, as it was the ws4sql binary (Line 9).

The rest of the lines in this example are standard Docker.

#### Important

* It's important to use `--user`, otherwise ws4sql will start with root privileges! You don't want this, as it 
  creates files; for example, backups may potentially overwrite some file or wreak havoc in unpredictable ways (which 
  will actually be _very_ predictable, but only after they happen).

#### Caveats

* You have to reference database and companion files that are in the directory mapped to `/data` as they were in `/data`;
* The path for the database file should be absolute, i.e. `/data/...`.
