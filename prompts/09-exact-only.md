# Prompt 09 — Exact-only scan mode

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 10.

## Read first

- `docs/architecture.md` (similarity, exact-only mode)
- `docs/use-cases.md` (UC-1, UC-3)
- `docs/adr/0003-exact-and-similar-are-separate-verdicts.md`
- `docs/adr/0010-similarity-algorithms.md`
- `docs/adr/0011-exact-only-scan-mode.md`
- `docs/patterns.md` (Strategy)

## Outcome

A scan can be limited to byte-identical files. Near-duplicates stay ungrouped. The default scan still builds similar groups.

## In scope

- Add an explicit exact-only argument to the scan use case. The zero value is the default: similarity runs as in ADR-0010. Do not treat a nil similarity strategy as the switch. The strategy stays a required dependency, and exact-only decides whether the use case calls it.
- When exact-only is set, do not call the similarity strategy. The session’s similar groups are empty. Exact groups are unchanged.
- Store the mode on the session and round-trip it through SQLite. An omitted or false value means similar files are included.
- Set prompt 09 to `done` in `docs/development-plan.md`.

## Out of scope

`POST /scans`, the review-page control, previews, a new fingerprint, and any change to the ADR-0010 thresholds. Prompt 10 places the switch on the action that starts the scan and passes `exactOnly` through.

## Acceptance tests

Use the same kind of near-duplicate PNG fixture as prompt 07.

- `TestAcceptance_ExactOnlySkipsNearDuplicates`: two images that the default scan would put in one similar group produce zero similar groups and zero exact groups. The similarity port is not called. A test double that fails the test when called is the way to show that.
- `TestAcceptance_DefaultScanStillGroupsSimilar`: the same two images, with exact-only left at the zero value, still produce one similar group and no delete recommendation.
- `TestAcceptance_ExactOnlyKeepsByteIdentical`: two files with the same bytes still produce one exact group and zero similar groups.
- `TestAcceptance_ExactOnlyRoundTrip`: saving the session and loading it returns the exact-only flag as it was stored. A session stored without the flag loads as similar-files-included.

The prompt 07 acceptance tests still pass.

## Patterns

Strategy. Exact-only chooses whether the scan calls the strategy. It is not a second similarity algorithm.
