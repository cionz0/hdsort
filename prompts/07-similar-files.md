# Prompt 07 — Similar files

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 08.

## Read first

- `docs/architecture.md` (similarity)
- `docs/use-cases.md` (UC-3)
- `docs/adr/0003-exact-and-similar-are-separate-verdicts.md`
- `docs/patterns.md` (Strategy)

## Outcome

Scans can attach similar groups. Those groups never carry a delete recommendation, and byte-identical files stay in exact groups only.

## First step, before code

Write `docs/adr/0010-similarity-algorithms.md` that cites the algorithms you will implement and states the threshold. ADR-0009 is already the dry-run decision.

Allowed citations:

- Images: Christoph Zauner, “Implementation and Benchmarking of Perceptual Image Hash Functions,” 2010, for pHash. Average hash is allowed when you cite the description published with Johannes Buchner’s ImageHash library and record the distance threshold in the ADR.
- Plain text: Moses Charikar, “Similarity Estimation Techniques from Rounding Algorithms,” 2002 (simhash). Record the bit length and the Hamming-distance threshold in the ADR.

Decode JPEG, PNG, and GIF with the Go standard library. Text similarity covers `.txt` and `.md` only. PDF and Office documents wait for a later ADR that cites an extractor. Do not invent a hash.

## In scope

A similarity Strategy used after exact grouping. Wire it into the scan use case. Persist similar groups on the session.

## Out of scope

Changing the keeper policy. Recommending a delete inside a similar group. HTTP.

## Acceptance tests

Build tiny fixtures in the test (a solid-color PNG and a one-bit-different PNG, plus two text files).

- `TestAcceptance_SimilarImagesGroupWithoutRecommendation`: the cited image hash groups them, content IDs differ, and the group has no delete recommendation.
- `TestAcceptance_IdenticalImagesAreExactOnly`: the same PNG bytes twice produce one exact group and zero similar groups.
- `TestAcceptance_SimilarTextGroupsWithoutRecommendation` for two `.txt` files within the ADR threshold.
- `TestAcceptance_DistantTextDoesNotGroup` for two `.txt` files past the threshold.
- `TestAcceptance_UserCanStillMarkSimilarForDelete`: storing an explicit delete decision on a similar-group member succeeds. The group’s recommendation stays empty.

Update the UC-3 diagram if the algorithms you cited add a step the current diagram does not show. Add the ADR to `docs/adr/README.md`.

## Patterns

Strategy. The scan use case depends on the strategy interface.
