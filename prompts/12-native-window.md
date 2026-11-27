# Prompt 12 — Native review window

Implement this prompt only. Stop when its acceptance tests pass. Do not start a later prompt.

## Read first

- `docs/architecture.md` (containers, HTTP surface, synced-folder hint, preview size limits)
- `docs/use-cases.md` (UC-1 through UC-4 and UC-7)
- `docs/adr/0002-loopback-web-ui-in-one-go-binary.md`
- `docs/adr/0007-staged-content-identity.md`
- `docs/adr/0009-dry-run-on-the-review-page.md`
- `docs/adr/0010-similarity-algorithms.md`
- `docs/adr/0011-exact-only-scan-mode.md`
- `docs/adr/0012-release-builds-enable-cgo.md`
- `docs/patterns.md` (Ports and Adapters, Dependency Rule, Pipes and Filters)

## Outcome

`hdsort` opens a native window. The scan runs beside that window. The window shows a progress bar, the file it is reading, a queue, and the files already processed, each with its size. Exact groups show the existing keeper suggestion, the size of every copy, and a thumbnail when the file is a JPEG, PNG, or GIF. Similar groups show thumbnails and sizes and do not suggest a delete.

## ADR first

Do this before any window code. A GUI toolkit is not in `docs/patterns.md` yet.

1. Add ADR-0013. Set ADR-0002 to Superseded by ADR-0013. Restate the parts that stay: one binary per OS/architecture, cgo enabled on the target OS (ADR-0012), no cloud API, release tags follow Semantic Versioning. Replace the browser, the capability URL, and the loopback review server as the product UI.
2. Cite Gio in `docs/patterns.md` before importing it: Elias Naur and the Gio authors, Gio, immediate-mode GUI for Go. <https://gioui.org>. Record the module version in the ADR consequences when `go.mod` gains it.
3. Name the package: `internal/adapters/gui` draws the window. The progress events and the view model live in `internal/app`, which does not import Gio. `internal/domain` stays the standard library only.

Cgo stays on. ADR-0012 already builds each archive on its own OS. Do not set `CGO_ENABLED=0`.

## In scope

- `cmd/hdsort` opens the window. It does not listen, does not print a capability URL, and does not launch a browser. `Version()` stays. Tests do not open a display; they pass a fake window runner, the same kind of boundary fake as the folder dialog.
- The scan use case emits progress while it runs. The window goroutine only paints the latest model. Walking, hashing, SQL, and Trash run off that goroutine.
- Listing the folder shows an indeterminate bar and a count of paths found. When the walk finishes, readable files are the queue. Each queued row shows the path and the size in bytes. A symlink or a dataless file goes straight to the processed list with its skip reason and is not hashed.
- While hashing, the status line shows the path, the size, and the stage from ADR-0007: size, partial (first and last 64 KiB, only for a file of at least 128 KiB), or full BLAKE3. The bar is processed / queued. A finished file leaves the queue and appears in the processed list with its size and an outcome: unique, exact duplicate, or waiting for the similarity pass.
- With exact-only off, a second bar runs during similarity: compared / candidates. Exact groups are already visible during that pass. Exact-only skips the second bar and leaves similar groups empty. The window has the exact-only control on the action that starts the scan.
- An exact group shows the keeper recommendation that already exists: library survivor, plain name, or newer modification time as a tie-break, with the existing reason text. It also shows each copy’s size and the total size of the suggested deletes. A tie still has no recommendation; the sizes are visible anyway. File size is not a new keeper reason and does not change `DefaultKeeper`.
- JPEG, PNG, and GIF copies show a preview. At most 1 MiB, the preview is the image itself. Above that, it is a JPEG no wider than 320 pixels, decoded and encoded with the standard library. Other types have no thumbnail.
- A similar group shows path, size, and the image preview when the type has one. It has no delete suggestion.
- The synced-folder banner stays, with the same sentence, when a path segment of the root is `CloudStorage`, `Mobile Documents`, `Dropbox`, or `Google Drive`.
- Dry run and Apply stay separate. Dry run shows the plan and the sentence “Nothing was changed.” and writes nothing. Apply reports the path currently passed to Trash.
- Keep, delete, and skip still go through the existing decision use case.

Update `docs/architecture.md` and `docs/use-cases.md` in the same change: the container is the native window, and the sequence diagrams name that window. Prose and Mermaid stay in agreement. Update the README run section so `./hdsort` opens the window.

Leave the HTTP adapter and its tests in the tree. They are no longer how the owner reviews a scan.

## Out of scope

A new keeper rank. A delete suggestion for similar files. A new similarity algorithm. Deleting the HTTP package. Code signing. A new release artifact name.

## Acceptance tests

Drive the view model and the progress sink. Use a temporary directory. Do not open a window and do not require a display.

- `TestAcceptance_ScanReportsProgressBeforeReturn`: the test receives one event that carries a fixture path and its size while `Scan` has not returned.
- `TestAcceptance_ProgressShowsQueueAndProcessed`: after listing, each readable file is queued with its size; after it is hashed it is processed and no longer queued. A symlink is processed with reason symlink and is never queued.
- `TestAcceptance_ProgressNamesHasherStage`: a file of at least 128 KiB produces a partial stage and, when the ends match another file, a full stage. A smaller file produces a full stage and no partial stage. The content ID is unchanged.
- `TestAcceptance_ExactGroupShowsKeeperAndSizes`: a plain-name group shows the keeper path, the suggested delete path, both sizes, the reason, and a freed total equal to the suggested delete’s size.
- `TestAcceptance_TiedGroupShowsSizesWithoutRecommendation`: equal modification times and no other winner leave the recommendation absent and still show both sizes.
- `TestAcceptance_ImageGroupShowsThumbnail`: a PNG wider than 320 pixels yields a JPEG no wider than 320. A `.txt` member has no thumbnail.
- `TestAcceptance_SimilarGroupHasNoDeleteSuggestion`: the similar group shows both paths and both sizes and no recommendation.
- `TestAcceptance_SyncedRootShowsBanner` for a root with a `Dropbox` segment, and the banner is absent for a root without one of the four segments.
- `TestAcceptance_DryRunShowsNothingChanged`: the model includes the delete path and “Nothing was changed.” The file is still in place.
- `TestAcceptance_ApplyNamesFileBeingTrashed`: the test observes the duplicate path while `Apply` has not returned. The file ends in the temporary Trash.
- `TestAcceptance_CommandStartsWindow`: the composition root runs the window runner and does not bind a port.

`go test ./...` and `go build -o /tmp/hdsort ./cmd/hdsort` both succeed with cgo enabled. On macOS and Linux that needs the host C compiler.

## Patterns

Gio is the immediate-mode shell, added only through ADR-0013 and the catalog citation. The view model is the port the window paints. Progress is reported from the existing scan stages. Keeper ranking stays Strategy as it is.
