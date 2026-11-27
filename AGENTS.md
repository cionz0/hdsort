# Agent notes

This file is for an agent session that implements hdsort. The product behavior is specified in `docs/`. Do not invent a parallel description here.

1. Read `docs/development-plan.md` and take the first prompt whose status is `not started`.
2. Read that prompt and the documents it names.
3. Implement only that prompt. Its `TestAcceptance_*` tests are part of the definition of done. The prompt is finished only when those tests and the tests from earlier prompts pass (`go test ./...`).
4. Update diagrams and prose when the behavior they describe changes.
5. Set that prompt’s row to `done`.
6. If the prompt closed with no errors, warnings, open questions, or leftover work, commit (README changes included), push, and tag that commit with the next SemVer version. Update MemPalace when the chat decided something the repository does not record. Then stop. Do not start the next prompt.

Patterns you may use are listed in `docs/patterns.md`. A pattern that is not listed needs an ADR and a citation before any code uses it.
