# Prompt 02 — Domain model

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 03.

## Read first

- `docs/architecture.md` (vocabulary, keeper rank names, plan checks)
- `docs/use-cases.md` (UC-2, UC-4, UC-5)
- `docs/adr/0003-exact-and-similar-are-separate-verdicts.md`
- `docs/adr/0005-layout-proposals-stay-inside-the-scan-root.md`
- `docs/adr/0006-acceptance-tests-are-the-specification.md`
- `docs/patterns.md` (functional core, Strategy)

## Outcome

Pure domain types and checks, with no filesystem or database imports. Policies that need I/O stay as data in, data out.

## In scope

Types from the vocabulary table: content ID, file ref, exact group, similar group, library entry, recommendation, decision, layout move, session.

Reason codes are the closed set `library_survivor`, `plain_name`, `newer_mtime_tiebreak`. An absent recommendation is a distinct state, not an empty reason string.

`internal/domain` imports only the standard library.

## Out of scope

Walking, hashing, SQL, HTTP, the ranking rules themselves (prompt 04), and any similarity algorithm (prompt 07).

## Acceptance tests

Table-driven tests in `internal/domain`.

- `TestAcceptance_ExactGroupNeedsTwoMembers`: constructing an exact group with fewer than two members returns an error.
- `TestAcceptance_SimilarGroupHasNoRecommendation`: a similar group cannot store a delete recommendation.
- `TestAcceptance_DecisionRequiresExplicitChoice`: a decision whose choice is not keep, delete, or skip is rejected.
- `TestAcceptance_UnconfirmedDeleteRejected`: a plan that trashes a path with no delete decision is rejected.
- `TestAcceptance_LastCopyRejected`: a plan that trashes every copy of a content ID that has no library entry is rejected.
- `TestAcceptance_LastCopyAllowedWhenLibraryKeepsIt`: the same plan is accepted when a valid library entry exists for that content ID.
- `TestAcceptance_MoveOutsideRootRejected`: a layout move whose destination escapes the scan root (`..` after `filepath.Rel`) is rejected.
- `TestAcceptance_SessionStatus`: a new session is `open`. The only transition this prompt implements is `open` to `applied`, and only through the apply-result function. A second apply on an `applied` session returns an error.

## Patterns

Functional core. Plan checks are pure functions. Do not introduce a pattern outside `docs/patterns.md`.
