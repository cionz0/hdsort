# Prompt 05 — Library, reconcile, and match

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 06.

## Read first

- `docs/architecture.md` (reconcile, library match, vocabulary)
- `docs/use-cases.md` (UC-6, and the library paragraphs of UC-2)
- `docs/adr/0004-trash-before-index-commit.md` (what a library entry means; do not implement Trash here)
- `docs/adr/0007-staged-content-identity.md`
- `docs/adr/0008-library-location.md`
- `docs/patterns.md` (Repository, SQLite WAL, OS data directories)

## Outcome

A SQLite library the scan can reconcile and match. Sessions are stored. Applying deletes is still prompt 06: this prompt only reads and updates library rows the tests insert directly.

## In scope

- `LibraryRepository` and `SessionRepository` ports in `internal/app`, SQLite adapter in `internal/adapters/sqlite`.
- Driver: a SQLite implementation that works with `database/sql` and does not require CGO, because ADR-0002 builds with `CGO_ENABLED=0`. `modernc.org/sqlite` is the expected module if its documentation still describes a pure-Go `database/sql` driver. If you choose another module, record the reason in a new ADR. Enable WAL.
- Open helper that resolves the ADR-0008 path, plus an open option that takes an explicit file path. Tests use only the explicit path.
- Reconcile: inside the scan root, missing file or changed content ID marks the entry stale; unchanged file refreshes metadata. Entries outside the root are not modified.
- Match: valid library entries participate in keeper ranking via the prompt 04 function. Stale entries do not.
- Persist the open session, including groups and recommendations.

## Out of scope

Trash, apply, HTTP, the default data directory’s real files (never open the developer’s library in a test).

## Acceptance tests

Temporary SQLite file and temporary directories.

- `TestAcceptance_ReconcileMissingMarksStale`
- `TestAcceptance_ReconcileChangedContentMarksStale`
- `TestAcceptance_ReconcileUnchangedRefreshesMetadata`
- `TestAcceptance_ReconcileIgnoresEntriesOutsideRoot`
- `TestAcceptance_StaleEntryDoesNotFormADuplicate`: one remaining file with a stale library row is not an exact group.
- `TestAcceptance_TwoCopiesRemainAGroupAfterStaleOriginal`: two in-scan copies still form one exact group, and the keeper is not the stale path.
- `TestAcceptance_MatchOutsideLibrarySurvivor`: the recommendation’s keeper is the outside library path and the reason code is `library_survivor`.
- `TestAcceptance_SessionRoundTrip`: saving an open session and loading it returns the same groups, recommendations, and root.

## Patterns

Repository. The reconcile use case orchestrates the repository and the filesystem port. Matching reuses the domain keeper Strategy.
