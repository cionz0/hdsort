# Prompt 17 — Progress paths stay inside the scan root

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-1, UC-4), and `prompts/14-readable-review.md`.

Prompt 14 is already in effect: paths drawn in the list and in the columns are relative to the chosen folder. `DisplayPath` leaves a path outside that folder unchanged.

## Outcome

The progress bar uses the same relative path. While a file is being read, the label is the path relative to the scan root, then the hasher stage, as in `notes/budget.pdf · full`. While a file is moving to Trash, the label is `Trash ` plus that relative path. “Listing the folder…”, “Scanning”, and “Comparing similar files” stay as they are, because they do not name a file.

The path stored on the progress event and the path passed to Trash stay absolute. Only the label changes. A path outside the scan root stays absolute.

## Acceptance tests

- `TestAcceptance_ProgressLabelIsRelativeToTheScanRoot`: with root `/docs`, a full-hash event for `/docs/notes/budget.pdf` labels `notes/budget.pdf · full`. With root `/backup`, trashing `/backup/Copy of photo.png` labels `Trash Copy of photo.png`. A path outside the root stays absolute.
- Update `TestAcceptance_RunningActionDisablesItsControl` so it passes the scan root into the slot and expects those relative labels. The disable rules do not change.

## Notes

Do not change which button is disabled, the accent rule, Open, or the release archive names. Update the Review window prose where it describes the progress label.
