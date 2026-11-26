# Prompt 04 — Keeper policy

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 05.

## Read first

- `docs/architecture.md` (keeper policy, generated-copy rules, reason codes)
- `docs/use-cases.md` (UC-2)
- `docs/adr/0003-exact-and-similar-are-separate-verdicts.md`
- `docs/patterns.md` (Strategy)

## Outcome

The default keeper Strategy fills a recommendation on an exact group, or leaves the recommendation absent when the documented signals tie.

## In scope

A pure function. Inputs are the exact-group members and, when prompt 05 has not landed yet, an optional library entry passed in by the caller. This prompt may accept that entry as a function argument so the policy stays pure. Prompt 05 will supply it from SQLite.

Generated-copy rules are exactly the three rules in `docs/architecture.md`. Version-looking names are not a rule.

## Out of scope

Persisting the recommendation, HTTP, and changing the rank order.

## Acceptance tests

Table-driven `TestAcceptance_KeeperRecommendation` rows:

| Situation | Keeper | Reason code |
| --- | --- | --- |
| `a.jpg`, `a (1).jpg`, `Copy of a.jpg`, same bytes | `a.jpg` | `plain_name` |
| `photo.jpg` and `photo conflicted copy.jpg` | `photo.jpg` | `plain_name` |
| two plain names, different modification times | the newer path | `newer_mtime_tiebreak` |
| two plain names, equal modification times | none | recommendation absent |
| library entry outside the scan, plus two in-scan copies | the library path | `library_survivor` |
| library path inside the scan, same content ID | the library path | `library_survivor` |
| `report_v1.pdf` and `report_final.pdf`, same bytes, `final` newer | `report_final.pdf` | `newer_mtime_tiebreak` |
| `report_v1.pdf` and `report_final.pdf`, different bytes | no group, so the policy is not called | — |

Also:

- `TestAcceptance_LibraryFileIsNotAMove`: the recommendation for an outside library survivor contains no layout move.
- `TestAcceptance_ReasonTextNamesTheTieBreak` when the code is `newer_mtime_tiebreak`: the human-readable reason contains the words that it was a tie-break. Assert that phrase, not the whole sentence.

## Patterns

Strategy, with one concrete policy: the default rank above. A second policy needs an ADR.
