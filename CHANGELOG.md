## v0.16.5
*2025-08-06, Venice*

- SQLite 3.50.3
- Library updates

## v0.16.4
*2025-04-21, Venice*

- SQLite 3.49.1
- Library updates
- Fixes for linter warnings

## v0.16.3
*2024-11-13, Venice*

- Library updates

## v0.16.2
*2024-06-19, Venice*

- SQLite 3.46.0
- Library updates

## v0.16.1
*2024-04-18, Venice*

- SQLite 3.46.0
- Library updates (incl. security fixes upstream)

## v0.16.0
*2024-02-17, Venice*

- #38: In a map resultset, keys are now in the same order as in the query
- #41: Query parameters as arrays
- #42: Remove encryption framework (**breaking change**)
- #43: Option to return results as a list, instead of a map
- Go 1.22 and SQLite 3.45.1

## v0.15.3
*2024-02-09, Venice*

- CI with Github Actions
- Switched to trunk based workflow
- Library updates (incl. security fixes upstream)

## v0.15.2
*2023-12-03, Venice*

- Library updates (incl. security fixes upstream)

## v0.15.1
*2023-10-06, Venice*

- Library updates
- First version of [`sqliterg`](https://github.com/proofrock/sqliterg), the "spiritual successor" to `ws4sqlite`.

## v0.15.0
*2023-05-07, Windhoek*

- Issue #29: Allow different path for db companion YAML file
- Issue #28: Rework build system
- Issue #27: Add support for linux/S390x
- Documentation regarding #33 (Dockerfile: allow specifying UID/GID with which ws4sqlite starts)
- Library updates

## v0.14.2
*2023-4-21, Busan*

- Issue #25: Rebase docker image(s) on distroless/static-debian11
- Issue #26: When checking file/dir existence, manage all errors
- Some minor improvements to build
- Library updates

## v0.14.1
*2023-4-4, Beppu*

- SQLite 3.40.2
- Library updates

## v0.14.0
*2023-2-28, Fes*

- SQLite 3.40.1
- Issue #21: Feature: multiple maintenance tasks
- Issue #22: Enhancement: keep the db connection open
- Issue #24: Small fixes to commandline args parsing
- Library updates

## v0.13.0
*2023-2-8, Venice*

- Issue #16: Feature: specify a custom error code for Not Authorized
- Issue #19: Feature: backup at each startup
- Issue #20: Feature: ability to perform statements during maintenance
- Issue #15: Out-of-order startup reporting of server configs
- Library updates
- Go 1.20

## v0.12.7
*2023-1-26, Venice*

- Issue #14: "Method not allowed" when calling OPTION in CORS preflight

## ~~v0.12.6~~
*2023-1-26, Venice*

- Issue #13: Middlewares activation is not correct
- Library updates

**IMPORTANT**: this version is broken, see #14

## v0.12.5
*2023-1-14, Venice*

- SQLite 3.40.1
- Library updates

## v0.12.4
*2022-12-19, Venice*

- Issue #11: runtime error: slice bounds out of range [:-1]
- Issue #12: Hardcoded path in error message
- SQLite 3.40.0
- Library updates

## v0.12.3
*2022-11-24, Hội An*

- SQLite 3.39.4
- Library updates

## v0.12.2
*2022-09-12, Venice*

- SQLite 3.39.3
- Library updates

## v0.12.1
*2022-08-12, Pulau Mabul*

- SQLite 3.39.2
- Library updates

## v0.12.0
*2022-07-29, Kota Kinabalu*

- SQLite 3.38.5
- Embedded Web Server, [docs here](https://germ.gitbook.io/ws4sqlite/documentation/web-server)
- Library updates

## v0.11.4

- SQLite 3.38.3
- Library updates
- Ditch 7z format for distribution, use .tar.gz & zip instead (according to the OS)

## v0.11.3

- Any file extension is now allowed [addresses #4]
- Migration from mattn/go-sqlite3 to modernc.org/sqlite
  - SQLite 3.38.2 
  - CGO is not required, allowing cross-compilation
  - Sequential access to db is always enforced; previously it wasn't in some cases, but they where
    rare, and this scheme is more consistent and not much slower
  - Read-only mode is enforced via query_only pragma
    - This closes an inconsistency: formerly, the engine was configured to expect an immutable file,
      but the same file could be opened concurrently by a non-readonly connection
    - It also makes it possible to provide a maintenance plan for read-only databases
- In crypgo dependency, replaced DataDog/zstd with klauspost/compress
  - Complete removal of CGO usage
- Usage of Go 1.18 
- New target (`make zbuild-all`) to cross-compile binaries
  - New targets linux/riscv64, windows/arm64 and freebsd/amd64 
- Unified docker builds for AMD64, ARM and ARM64 (under the same tag)
- Several updates to docs, libs and minor refactorings of the code
- [#5] Better (but not complete) support of make under windows
