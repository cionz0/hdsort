# ADR 0008: Library location

- Status: Accepted
- Date: 2026-11-26

## Context

The survivor index has to outlive a single scan and has to stay off the user’s documents folder, so a later scan does not treat the index as one more file to organize. The location conventions differ by OS.

## Decision

The library is one SQLite file in the per-user application data directory:

| OS | Directory | Source |
| --- | --- | --- |
| macOS | `~/Library/Application Support/hdsort/library.sqlite` | Apple File System Programming Guide, Application Support |
| Linux | `$XDG_DATA_HOME/hdsort/library.sqlite`, or `~/.local/share/hdsort/library.sqlite` when `XDG_DATA_HOME` is unset | XDG Base Directory Specification |
| Windows | `%AppData%\hdsort\library.sqlite` | `FOLDERID_RoamingAppData` |

Sessions live in the same directory. The scan walk skips this directory.

SQLite runs in WAL mode. The process is the only writer.

## Consequences

- The driver is `modernc.org/sqlite` v1.63.0. Its package comment describes a CGo-free `database/sql` driver (“Package sqlite is a sql/database driver using a CGo-free port of the C SQLite3 library.”), which matches ADR-0002 (`CGO_ENABLED=0`). `Open` sets `journal_mode=WAL` and returns an error when the mode is not `wal`.
- Sessions are rows in the same SQLite file as the library. That file is the one in the table above, so sessions stay in that directory.
- Tests pass an explicit temporary path into the repository. Tests do not read the developer’s real library.
- A future “library path” flag may override the default. Until that ADR exists, the table above is the only location.
