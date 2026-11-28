# Prompt 22 — Last-copy warning in the preview

Read `docs/architecture.md` (Review window, Sessions and apply), `docs/use-cases.md` (UC-2, UC-4), `docs/adr/0015-confirm-last-copy.md`, and `prompts/21-confirm-last-copy.md`.

Prompt 21 is already in effect: Apply refuses to trash the last copies of a file the library does not keep until the owner confirms. That confirmation is one note above the list and one button, `Trash the last copies`, for the whole plan. This prompt replaces that presentation.

## Outcome

The warning sits in the compare pane, on the same surface as the previews of the selected set. It does not sit in the note above the list. It is drawn with the accent, so it is not a quiet line of secondary text. It does not show the content ID. It lists each copy selected for deletion as its path relative to the scan root, one path per line.

The button on that warning confirms only the selected copies of that file. It does not confirm any other file. The outline marks each set that still needs this confirmation, so the owner can open it.

Apply trashes nothing while any such file is unconfirmed. When the owner confirms the last one, Apply continues without a second click: it trashes the confirmed copies, it does not upsert those content IDs, and it marks the session applied. A file the library already keeps does not show the warning. A trash path with no delete decision is still rejected. Changing keep, delete, or skip for one of that file’s copies clears its confirmation.

Remove the global note and the single `Trash the last copies` button from prompt 21.

Add ADR-0016 for this per-file confirmation. Do not supersede ADR-0004. ADR-0015 still detects the last copy; this prompt replaces only how the owner confirms it.

## Acceptance tests

Use a temporary directory and the Freedesktop trash adapter. Do not open a window.

- `TestAcceptance_LastCopyWarningIsPerFile`: two files, each with both copies marked delete, and an empty library. Confirming one file does not trash the other. The unconfirmed file’s copies remain. Nothing is written to the library.
- `TestAcceptance_ApplyProceedsWhenEveryLastCopyIsConfirmed`: confirming each of those files trashes all four copies, writes no library entry, and marks the session applied.
- `TestAcceptance_ChangingAChoiceClearsThatConfirmation`: after a file is confirmed, changing one of its choices makes Apply wait again. Its copies remain.
- `TestAcceptance_LibraryCopyNeedsNoWarning`: a valid library entry for that content ID lets Apply trash the in-folder copies with no extra confirmation.

`go test ./...` and `go vet ./...` pass with cgo enabled.

## Notes

Do not change the keeper rank, the folder-trash action, Open, the radios, or the release archive names. Update the Review window prose, the UC-4 prose, and both sequence diagrams in the same change. The accent sentence names this warning.
