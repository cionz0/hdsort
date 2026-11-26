# Prompt 06 — Plan, apply, and layout

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 07.

## Read first

- `docs/architecture.md` (sessions and apply, layout)
- `docs/use-cases.md` (UC-4, UC-5)
- `docs/adr/0004-trash-before-index-commit.md`
- `docs/adr/0005-layout-proposals-stay-inside-the-scan-root.md`
- `docs/patterns.md` (Repository, Strategy, Freedesktop Trash specification)

## Outcome

The owner’s decisions can be stored, checked, and applied: Trash first, library upsert second. The default layout strategy proposes no moves.

## In scope

- Record keep, delete, and skip decisions on a session.
- Apply use case: domain plan check, then Trash port, then library upsert of survivors at their final paths.
- On any Trash error, leave the library bytes unchanged (assert with a row count or a content-ID query from before the call) and leave the session `open`.
- Linux Trash adapter implements the Freedesktop Trash specification against a directory the caller passes (so tests set `XDG_DATA_HOME` or an equivalent constructor argument). Do not `os.Remove` the original.
- macOS and Windows adapters live in build-tagged files and call the platform Trash API. Unit-test the input validation they share. Do not claim `go test` opens Finder or Explorer.
- Default layout Strategy returns an empty move list.
- A confirmed move whose destination stays inside the root is performed during apply, after Trash succeeds, and the library path is the destination. Performing the move uses `os.Rename` inside the root only after the domain check from prompt 02.

## Out of scope

A photo-date or by-type layout strategy. HTTP. Similarity.

## Acceptance tests

- `TestAcceptance_ApplyTrashesThenIndexes`: after apply, the deleted file is in the test Trash and the keeper is a library row. The original path of the deleted file is gone.
- `TestAcceptance_ApplyStopsWhenTrashFails`: a Trash port double that returns an error is allowed in this one test, because the failure is the OS boundary. The library row count is unchanged and the session is `open`.
- `TestAcceptance_ApplyRejectsUnconfirmedDelete`: no Trash call, session stays `open`. Count calls on the port if you need to show it was not called.
- `TestAcceptance_DefaultLayoutIsEmpty` on a fixture with nested folders.
- `TestAcceptance_ConfirmedMoveStaysInsideRoot`: a confirmed move relocates the survivor and the library path matches the destination.
- `TestAcceptance_UnconfirmedMoveDoesNotRun`.
- `TestAcceptance_FreedesktopTrashLayout`: the file lands in the Trash `files` directory and an `info` entry exists, per the Trash specification.

## Patterns

Strategy for the layout policy (one implementation: empty). Repository for the session and the library. The Trash adapter is a driven adapter.
