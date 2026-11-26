# Prompt 09 — Scan start and previews

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 10.

## Read first

- `docs/architecture.md` (HTTP surface, opaque identifiers)
- `docs/use-cases.md` (UC-1)
- `docs/adr/0002-loopback-web-ui-in-one-go-binary.md`
- `docs/patterns.md` (Ports and Adapters, OWASP Path Traversal)

## Outcome

The page can start a scan and can show a preview of a file that belongs to that session. Previews never take a filesystem path from the client.

## In scope

- `POST /scans` with a JSON path starts UC-1 and returns the session id. An empty JSON object asks the folder-dialog port. Tests pass a fake dialog that returns a fixture directory. The native dialog adapter is wired in `cmd/hdsort` for the host OS and is not what `go test` clicks.
- File ids are opaque values the session assigns. `GET /scans/{id}/files/{fileID}/preview` returns the bytes.
- A request that includes a `path` query or a raw filesystem path instead of a file id is 400.
- A file id from another session is 404.
- For JPEG, PNG, and GIF, the preview response may be the original bytes when the file is at most 1 MiB. Larger images get a JPEG thumbnail no wider than 320 pixels, decoded and encoded with the standard library. Other types return the original bytes with `Content-Disposition: attachment` only when the file is at most 1 MiB; larger non-images return 415. Record these limits in `docs/architecture.md` next to the preview route, with the diagram unchanged unless the route shape changes.

## Out of scope

A new similarity algorithm. Cloud downloads. Changing keeper ranking.

## Acceptance tests

- `TestAcceptance_PostScanCreatesSession` on a fixture with one exact pair. The JSON contains one exact group.
- `TestAcceptance_EmptyPostUsesFolderPicker` with the fake picker.
- `TestAcceptance_PreviewByIDReturnsBytes` for a small PNG.
- `TestAcceptance_PreviewRejectsRawPath`: `?path=` or a body path is 400 and the bytes are not the file.
- `TestAcceptance_PreviewUnknownIDIs404`, including an id that exists on a different session.
- `TestAcceptance_LargeImageIsThumbnail`: a PNG wider than 320 pixels comes back as a JPEG no wider than 320.

Update the preview row in `docs/architecture.md` so the size limits are part of the spec.

## Patterns

The folder dialog is a driven port. The HTTP adapter translates ids to the file ref the session already stored. It does not join client input onto a root path.
