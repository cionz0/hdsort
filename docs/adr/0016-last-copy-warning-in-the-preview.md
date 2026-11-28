# ADR 0016: Confirm the last copy in the preview

- Status: Accepted
- Date: 2026-11-28

## Context

ADR-0015 asks before Apply trashes the last copies of a file the library does not keep. The window asked once, above the list, for the whole plan. The owner could not see the warning next to the previews, and one confirmation covered every file.

## Decision

Each file that would lose its last copy has its own warning in the compare pane, on the same surface as that set’s previews. The warning uses the accent. It lists the copies selected for deletion as paths relative to the scan root, one per line, and does not show the content ID. The outline marks each set that still needs confirmation.

The button confirms only that file. Apply trashes nothing while any such file is unconfirmed. When the last one is confirmed, Apply continues without a second click. Changing keep, delete, or skip for one of that file’s copies clears its confirmation. A file the library already keeps has no warning.

## Consequences

- ADR-0004 stays Accepted. An unconfirmed last-copy plan is still rejected, and a trash path with no delete decision is still rejected.
- ADR-0015 still detects the last copy. This ADR replaces only the single global confirmation.
