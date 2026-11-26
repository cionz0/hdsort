# ADR 0004: Trash before the survivor index is updated

- Status: Accepted
- Date: 2026-11-26

## Context

The survivor index is the memory used by later scans. If the index is updated before the delete succeeds, a later scan will treat a file the user still has as already handled. If a delete uses unlink, a mistake is not recoverable from the operating system Trash.

## Decision

Applying a session does two steps, in this order:

1. Move every confirmed delete to the operating system Trash.
2. If every trash operation succeeded, upsert the survivors into the library.

If any trash operation fails, the library is left unchanged and the session stays unapplied.

A plan is rejected when a delete has no user confirmation, or when the plan would remove the last known copy of a content ID that is not already a library survivor.

The Trash port follows the Freedesktop Trash specification on Linux. macOS and Windows adapters use the platform Trash facility rather than `os.Remove`.

## Consequences

- Acceptance tests use a temporary directory that implements the Freedesktop layout, so the Linux adapter is tested on any OS by pointing the adapter at that directory.
- The macOS and Windows adapters are behind build tags. Their path-construction and error behavior are unit-tested; the real Finder or Explorer interaction is not part of `go test`.
- Undo is “put it back from Trash” while the OS still holds the file. hdsort records the trashed paths in the session so the UI can name them.
