# Prompt 20 — Trash the selected folder

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-2, UC-4), `docs/adr/0004-trash-before-index-commit.md`, and `prompts/19-folder-outline.md`.

Prompt 19 is already in effect: the left pane is an expandable folder outline. This prompt replaces one rule from that prompt. The disclosure control still only expands or collapses a directory. Clicking the directory name selects that directory. Clicking a set still selects the set.

## Outcome

When a directory is selected, the compare pane shows that folder and one action: trash the folder and everything inside it. The button uses the accent. Its label names the folder the window already draws, for example `Trash Calendar and its contents`. The scan root uses its base name.

The action calls the Trash port once with the cleaned absolute path of that directory. The port moves the directory. The use case does not walk the tree and does not unlink. Files the list does not show, including unique files, go with the directory. A sibling outside the directory stays on disk.

The directory must be the scan root or inside it. `PathInside` already treats the root as inside itself. A folder outside the scan root, including the single outside node from prompt 19, is refused. Nothing is moved.

This action is not Apply. It does not run the session plan, and it does not apply the last-copy check. The owner has confirmed this directory. The session stays open and unapplied. The library is not written. The next scan’s existing reconcile marks a missing path stale.

After a successful trash, drop every decision whose path is inside the directory. A decision outside it remains. Drop a set row only when every member is inside the directory. A set that still has a member outside keeps that member and is filed under the first remaining member’s directory. While the trash runs, the progress label is `Trash` plus the folder’s path relative to the scan root, and the folder button is disabled. A failed trash leaves the directory, the decisions, and the rows as they were.

Add ADR-0014 for this action. Do not supersede ADR-0004. Apply is unchanged.

## Acceptance tests

Use a temporary directory and the Freedesktop trash adapter pointed at another temporary directory. Do not open a window.

- `TestAcceptance_TrashFolderMovesTheDirectory`: `a/b/file.txt` and `a/keep.txt` move with `a`. `c/other.txt` remains. The trashed `a` contains both files.
- `TestAcceptance_TrashFolderRejectsOutsideTheScanRoot`: a directory that is not inside the scan root returns an error. Both directories stay in place.
- `TestAcceptance_TrashFolderDropsDecisionsInsideIt`: a decision inside the directory is gone, a decision outside it remains, the session is still open, and the library list is unchanged.
- `TestAcceptance_SetKeepsMembersOutsideTheTrashedFolder`: a set with one member inside the directory and one outside keeps the outside member and is filed under that member’s directory.

`go test ./...` and `go vet ./...` pass with cgo enabled.

## Notes

Do not change the keeper rank, the similar-group delete rule, Open, the radios, which files the scan lists, or the release archive names. Update the Review window prose and its sequence diagram, and UC-2, in the same change. The accent sentence also names this button.
