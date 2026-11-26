# ADR 0009: Dry-run is a review-page action

- Status: Accepted
- Date: 2026-11-26

## Context

Apply moves files to Trash and then updates the survivor library. The owner already confirms each delete, and still asked for a dry-run on the review page: a way to see the effect of the current decisions before anything is written.

## Decision

Dry-run and apply share one plan check (unconfirmed deletes and the last-copy rule).

Dry-run returns the operations apply would perform and stops there:

- paths that would move to Trash
- confirmed layout moves that would run
- library upserts that would be written, using each survivor’s final path

It does not call Trash, does not rename files, and does not write the library. The session stays `open`. A later Apply is still required to carry the plan out.

When the plan check fails, dry-run returns the same rejection apply would return, and still writes nothing.

The review page offers Dry run as its own control, separate from Apply. The report on that page includes the sentence “Nothing was changed.”

## Consequences

- The plan check stays a pure function in `internal/domain`. The use case either returns the report or, for Apply, runs the same plan through Trash and the library (functional core, imperative shell).
- Prompt 06 implements the use case. Prompt 08 puts the control and the report on the review page.
- Dry-run is not a second confirmation. Decisions already stored on the session are what both actions read.
