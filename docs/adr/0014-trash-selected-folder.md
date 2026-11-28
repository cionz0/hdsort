# ADR 0014: Trash a selected folder as one directory

- Status: Accepted
- Date: 2026-11-28

## Context

The review outline can select a directory, not only a duplicate set. The owner wants that directory, and everything inside it, to go to the operating system Trash. That tree includes files the scan never listed as duplicates. ADR-0004 still rejects an Apply plan that would delete the last known copy of a content ID, and it still updates the library only after the plan's deletes succeed.

## Decision

Selecting a directory offers one action: trash that directory and its contents. The use case calls the Trash port once with the directory's cleaned absolute path. It does not walk the tree and does not unlink.

The directory must be the scan root or inside it. A directory outside the scan root is refused, and nothing is moved.

This action is not Apply. It does not run the session plan and does not apply the last-copy check. The session stays open. The library is not written. The next scan's reconcile marks a missing path stale. Decisions whose path is inside the trashed directory are dropped. A decision outside it remains.

The accent fills this button as well as Apply and the exact-folder action.

## Consequences

- ADR-0004 stays Accepted. Apply is unchanged.
- A unique file inside the selected directory is trashed with the directory.
- Undo is putting the folder back from Trash while the operating system still holds it.
