# Development plan

Each file in `prompts/` is one implementation session. Run them in order. A session implements the prompt it was given. The prompt is finished when its acceptance tests and the tests from earlier prompts pass (`go test ./...`).

Status below is the source of truth for “what is next.” The session that finishes a prompt sets that row to `done` and names the test package it added.

| Order | Prompt | Status | Outcome |
| --- | --- | --- | --- |
| 1 | [prompts/01-skeleton.md](../prompts/01-skeleton.md) | done (`cmd/hdsort`) | Go module, package layout, version command, test workflow |
| 2 | [prompts/02-domain-model.md](../prompts/02-domain-model.md) | done (`internal/domain`) | Groups, recommendations, plan checks |
| 3 | [prompts/03-identify-and-group.md](../prompts/03-identify-and-group.md) | done (`internal/app`) | Walk, staged BLAKE3, exact groups |
| 4 | [prompts/04-keeper-policy.md](../prompts/04-keeper-policy.md) | done (`internal/domain`) | Explainable keeper ranking |
| 5 | [prompts/05-library-and-match.md](../prompts/05-library-and-match.md) | done (`internal/app`) | SQLite library, reconcile, match |
| 6 | [prompts/06-plan-apply-layout.md](../prompts/06-plan-apply-layout.md) | done (`internal/app`) | Decisions, Trash, index commit, dry-run report, default layout |
| 7 | [prompts/07-similar-files.md](../prompts/07-similar-files.md) | done (`internal/app`) | Cited similarity strategy, no delete suggestion |
| 8 | [prompts/08-loopback-review.md](../prompts/08-loopback-review.md) | done (`internal/adapters/http`) | Loopback server, token, review page, dry-run control |
| 9 | [prompts/09-exact-only.md](../prompts/09-exact-only.md) | done (`internal/app`) | Exact-only scan mode; default scan still groups similar files |
| 10 | [prompts/10-scan-and-preview.md](../prompts/10-scan-and-preview.md) | done (`internal/adapters/http`) | Start a scan from the page, pass exactOnly, previews by file id |
| 11 | [prompts/11-release-binaries.md](../prompts/11-release-binaries.md) | done (`cmd/hdsort`) | Cross-compiled GitHub Release binaries |
| 12 | [prompts/12-native-window.md](../prompts/12-native-window.md) | done (`internal/app`) | Native window, scan progress, thumbnails, keeper and sizes |
| 13 | [prompts/13-review-by-folder.md](../prompts/13-review-by-folder.md) | done (`internal/app`) | One list grouped by folder, compare columns, system colors with light and dark fallbacks |
| 14 | [prompts/14-readable-review.md](../prompts/14-readable-review.md) | done (`internal/app`) | Duplicate sets only, relative paths, readable sizes, separated columns, system preview |
| 15 | [prompts/15-open-copy.md](../prompts/15-open-copy.md) | done (`internal/app`) | Open one copy in the default application |
| 16 | [prompts/16-quiet-system-colors.md](../prompts/16-quiet-system-colors.md) | done (`internal/app`) | Quiet system colors; accent only on committing actions |
| 17 | [prompts/17-progress-relative-path.md](../prompts/17-progress-relative-path.md) | done (`internal/app`) | Progress bar paths relative to the scan root |
| 18 | [prompts/18-choice-radios.md](../prompts/18-choice-radios.md) | done (`internal/app`) | Keep, Delete, and Skip as radios, with the suggestion selected |
| 19 | [prompts/19-folder-outline.md](../prompts/19-folder-outline.md) | done (`internal/app`) | Expandable folder outline in the left pane |
| 20 | [prompts/20-trash-selected-folder.md](../prompts/20-trash-selected-folder.md) | done (`internal/app`) | Trash the selected folder and everything inside it |
| 21 | [prompts/21-confirm-last-copy.md](../prompts/21-confirm-last-copy.md) | done (`internal/app`) | Confirm before trashing the last copy, naming each file |

## How to run a session

1. Read `README.md`, `docs/architecture.md`, `docs/use-cases.md`, `docs/patterns.md`, and the ADRs the prompt names.
2. Implement only that prompt.
3. Keep the acceptance tests from the prompt. They are part of the definition of done (ADR-0006). Finish only when those tests and the earlier ones pass (`go test ./...`).
4. Update the diagram and the prose if the behavior you implemented changed a document.
5. Set this table’s status for that prompt to `done`.
6. If the prompt closed with no errors, warnings, open questions, or leftover work, commit (README changes included), push, and tag that commit with the next SemVer version. Update MemPalace when the chat decided something the repository does not record. Then stop. Do not start the next prompt in the same session.

## Current product boundary

The decisions those prompts are not allowed to reopen are ADR-0002 through ADR-0008. A prompt that needs a different behavior adds a new ADR and sets the old one to Superseded.
