# Development plan

Each file in `prompts/` is one implementation session. Run them in order. A session implements the prompt it was given and stops when that prompt’s acceptance tests pass.

Status below is the source of truth for “what is next.” The session that finishes a prompt sets that row to `done` and names the test package it added.

| Order | Prompt | Status | Outcome |
| --- | --- | --- | --- |
| 1 | [prompts/01-skeleton.md](../prompts/01-skeleton.md) | done (`cmd/hdsort`) | Go module, package layout, version command, test workflow |
| 2 | [prompts/02-domain-model.md](../prompts/02-domain-model.md) | done (`internal/domain`) | Groups, recommendations, plan checks |
| 3 | [prompts/03-identify-and-group.md](../prompts/03-identify-and-group.md) | done (`internal/app`) | Walk, staged BLAKE3, exact groups |
| 4 | [prompts/04-keeper-policy.md](../prompts/04-keeper-policy.md) | not started | Explainable keeper ranking |
| 5 | [prompts/05-library-and-match.md](../prompts/05-library-and-match.md) | not started | SQLite library, reconcile, match |
| 6 | [prompts/06-plan-apply-layout.md](../prompts/06-plan-apply-layout.md) | not started | Decisions, Trash, index commit, dry-run report, default layout |
| 7 | [prompts/07-similar-files.md](../prompts/07-similar-files.md) | not started | Cited similarity strategy, no delete suggestion |
| 8 | [prompts/08-loopback-review.md](../prompts/08-loopback-review.md) | not started | Loopback server, token, review page, dry-run control |
| 9 | [prompts/09-scan-and-preview.md](../prompts/09-scan-and-preview.md) | not started | Start a scan from the page, previews by file id |
| 10 | [prompts/10-release-binaries.md](../prompts/10-release-binaries.md) | not started | Cross-compiled GitHub Release binaries |

## How to run a session

1. Read `README.md`, `docs/architecture.md`, `docs/use-cases.md`, `docs/patterns.md`, and the ADRs the prompt names.
2. Implement only that prompt.
3. Keep the acceptance tests from the prompt. They are the definition of done (ADR-0006).
4. Update the diagram and the prose if the behavior you implemented changed a document.
5. Set this table’s status for that prompt to `done`.
6. Stop. Do not start the next prompt in the same session.

## Current product boundary

The decisions those prompts are not allowed to reopen are ADR-0002 through ADR-0008. A prompt that needs a different behavior adds a new ADR and sets the old one to Superseded.
