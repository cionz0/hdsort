# ADR 0005: Layout proposals stay inside the scan root

- Status: Accepted
- Date: 2026-11-26

## Context

After duplicates are removed, the remaining files may sit in a messy folder. Reorganizing someone’s disk without a confirmation, or moving a file out of the folder they asked about, is a different product from the one decided in the design chat.

## Decision

Layout is a Strategy selected for a session. The default strategy proposes zero moves.

A move is stored on the session as a proposal. It is executed only after the user confirms that specific move. The destination must stay inside the scan root: `filepath.Rel(root, destination)` must not be `..` and must not start with `..` plus a separator. The check uses the cleaned absolute paths.

The default strategy is the one shipped first. A later strategy (for example photos by capture date) needs an ADR before it can propose moves, and it still waits for confirmation.

## Consequences

- A scan whose files are already arranged produces an empty move list.
- The library stores the survivor path after a confirmed move, not the path from before the move.
- Gathering every library entry into one canonical folder is a future command, not part of apply.
