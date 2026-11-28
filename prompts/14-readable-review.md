# Prompt 14 — A readable duplicate review

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-1, UC-2), and `docs/adr/0013-native-review-window.md`.

## Outcome

The list shows only duplicate sets. A file that is unique, a symlink, or a dataless placeholder leaves the list. Paths are relative to the chosen folder. Sizes are shown as B, KB, MB, or GB. Each copy is its own bordered column, and the Keep, Delete, and Skip buttons have a gap between them. A preview is the operating-system thumbnail when that call returns one, and otherwise the existing JPEG, PNG, or GIF thumbnail.

## Acceptance tests

- `TestAcceptance_UniqueFilesLeaveTheList`: a file processed as unique is not a row. After review, a unique file from the scan is absent and each duplicate path is on one group row.
- `TestAcceptance_SizeIsHumanReadable`: 0 is `0 B`, 1536 is `1.5 KB`, 39854 is `39 KB`.
- `TestAcceptance_PathIsRelativeToTheScanRoot`: a file inside the chosen folder is shown from that folder. A path outside it stays unchanged.
- `TestAcceptance_SystemPreviewUsedWhenPresent`: when the system preview returns bytes, those bytes are the preview, including for a file that is not a JPEG, PNG, or GIF.
- `TestAcceptance_ImagePreviewWhenSystemHasNone`: when the system preview returns nothing, a PNG still has a thumbnail.

## Notes

Decisions still store the cleaned absolute path. The relative path is only what the window draws. Linux has no system thumbnail without linking a toolkit, so a non-image there has no preview. Do not change the keeper rank, the similar-group delete rule, or the release archive names.
