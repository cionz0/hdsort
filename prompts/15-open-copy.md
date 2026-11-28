# Prompt 15 — Open one copy

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-2), and `docs/adr/0013-native-review-window.md`.

## Outcome

Each copy column has an Open control, with the same gap that already separates Keep, Delete, and Skip. Open asks the operating system to open that copy in its default application. It does not reveal the folder, record a decision, or trash the file. The path passed to the operating system is the cleaned absolute path, not the relative path drawn in the column.

One of the two actions the owner asked for is enough. This prompt opens the file. Revealing the containing folder is out of scope.

## Acceptance tests

- `TestAcceptance_OpenCopyAsksTheSystemForThatFile`: opening a copy calls the opener with that copy’s cleaned absolute path. The session’s decisions are unchanged.
- `TestAcceptance_OpenCopyReportsAFailure`: when the opener returns an error, the caller returns that error and still records no decision.

The opener is a port the use case does not own, the same kind of boundary as the folder dialog. The acceptance tests use a fake of that port. They must not launch Preview, Finder, or any other application.

## Notes

macOS uses `open` on the absolute path. Linux uses `xdg-open`. Windows uses the shell’s open verb. A failure is shown on the window’s existing note line. Do not change the keeper rank, which files stay in the list, the similar-group delete rule, or the release archive names. Update the Review window prose and its sequence diagram in the same change.
