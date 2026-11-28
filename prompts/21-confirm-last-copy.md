# Prompt 21 — Confirm the last copy

Read `docs/architecture.md` (Sessions and apply), `docs/use-cases.md` (UC-4), `docs/adr/0004-trash-before-index-commit.md`, and `docs/adr/0009-dry-run-on-the-review-page.md`.

Prompt 06 is already in effect: Apply rejects a plan that would trash every known copy of a content ID the library does not keep. The window shows that rejection as `plan trashes every copy of a content ID with no library entry` and a content ID.

## Outcome

That rejection is no longer the end of the action. Apply and Dry run still detect it, and neither writes anything until the owner confirms. The window then asks whether to trash those files anyway. It does not show the content ID. Each file is one line, the path relative to the scan root, so the owner can see which files they are. An accent button, `Trash the last copies`, carries the plan out.

Without that confirmation, Apply still returns the last-copy error, the files stay on disk, and the library is unchanged. With it, Apply trashes every copy of that content ID, does not upsert it, and marks the session applied. A plan that trashes a path with no delete decision is still rejected, confirmation or not. Dry run still writes nothing.

Add ADR-0015. Do not supersede ADR-0004. An unconfirmed last-copy plan stays rejected.

## Acceptance tests

Use a temporary directory and the Freedesktop trash adapter. Do not open a window.

- `TestAcceptance_LastCopyStillNeedsConfirmation`: both copies are marked delete and the library has no entry. Apply returns the last-copy error. Both files remain. The session stays open.
- `TestAcceptance_ConfirmedLastCopyTrashesEveryCopy`: the same plan, confirmed, moves both files to Trash, writes no library entry, and marks the session applied.
- `TestAcceptance_ConfirmationDoesNotSkipOtherChecks`: a trash path with no delete decision is still rejected when confirmation is set. The file remains.
- `TestAcceptance_LastCopyQuestionNamesThePaths`: the question lists each path relative to the scan root on its own line and does not contain the content ID.

`go test ./...` and `go vet ./...` pass with cgo enabled.

## Notes

Do not change the keeper rank, the folder-trash action, or the release archive names. CheckPlan still reports the last-copy error. Update the UC-4 prose and its sequence diagram, and the apply prose in `docs/architecture.md`, in the same change. The accent sentence also names this button.
