# Prompt 08 — Loopback review server

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 09.

## Read first

- `docs/architecture.md` (HTTP surface, synced-folder hint)
- `docs/use-cases.md` (UC-2, UC-3, UC-7, the banner paragraph)
- `docs/adr/0002-loopback-web-ui-in-one-go-binary.md`
- `docs/adr/0009-dry-run-on-the-review-page.md`
- `docs/patterns.md` (Ports and Adapters, `crypto/rand`, `crypto/subtle`, Secure Contexts)

## Outcome

A loopback HTTP server serves the review page and the decision routes. Tests use `httptest` plus the real session repository on a temporary SQLite file.

## In scope

- Listener helper used by `cmd/hdsort` and by tests. It binds `127.0.0.1` with port `0` when the caller asks for an ephemeral port. It refuses an address that is not loopback.
- Token of at least 128 bits from `crypto/rand`. Cookie `hdsort_token`, `HttpOnly`, `SameSite=Strict`, `Secure`, `Path=/`. `Authorization: Bearer` is accepted as well. Compare with `crypto/subtle`.
- `GET /?token=` sets the cookie and returns the review HTML.
- `GET /scans/{id}` returns JSON for an existing session.
- `POST /scans/{id}/decisions` records a decision.
- The review HTML shows exact groups and similar groups in two sections. It shows the recommendation reason when one exists, and it shows the synced-folder banner when a path segment of the root is `CloudStorage`, `Mobile Documents`, `Dropbox`, or `Google Drive`.
- The review page has a Dry run control, separate from Apply, that POSTs to `/scans/{id}/dry-run`. The HTML response is the review page with the report and the sentence “Nothing was changed.” A request with `Accept: application/json` gets that report as JSON with `changed` set to false. A rejected plan returns the same rejection as apply, still with nothing written.
- Missing token is 401. Wrong token is 401.

`cmd/hdsort` with no arguments starts the listener and prints the capability URL. Opening the browser may use the OS open command; tests do not assert that the browser launched.

## Out of scope

Starting a scan from the page (prompt 10) and previews. Wire `POST /scans/{id}/apply` and `POST /scans/{id}/dry-run` to the prompt 06 use cases so the page can finish UC-4 and UC-7.

## Acceptance tests

- `TestAcceptance_ListenerIsLoopback`: the helper’s address IP is loopback.
- `TestAcceptance_ListenerRejectsWildcard`: asking to bind `0.0.0.0` returns an error.
- `TestAcceptance_MissingTokenIs401` on `GET /scans/{id}`.
- `TestAcceptance_BearerTokenReadsSession`.
- `TestAcceptance_ReviewPageShowsExactAndSimilar`: the HTML contains the keeper path, a recommended delete path, and a similar-group member, in separate sections.
- `TestAcceptance_ReviewPageShowsSyncBanner` when the root contains a `Dropbox` path segment, and `TestAcceptance_ReviewPageOmitsSyncBanner` when it does not.
- `TestAcceptance_PostDecisionPersists` and a following GET returns that decision.
- `TestAcceptance_ApplyRouteTrashes`: `POST /scans/{id}/apply` with a confirmed delete performs UC-4 against the temp Trash and temp library.
- `TestAcceptance_ReviewPageOffersDryRun`: the review HTML contains a Dry run control whose action is the dry-run route, and a separate Apply control.
- `TestAcceptance_DryRunPageWritesNothing`: posting the dry-run route for a confirmed delete returns HTML that includes the delete path and the sentence “Nothing was changed.” The file is still at its original path, the library is unchanged, and the session is `open`. The JSON form of the same call has `changed` false and the same path.

## Patterns

HTTP adapter is the driving adapter. It calls use cases; it does not hash files or write SQL itself.
