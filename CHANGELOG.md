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
