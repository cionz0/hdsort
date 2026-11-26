# Agent notes

This file is for an agent session that implements hdsort. The product behavior is specified in `docs/`. Do not invent a parallel description here.

1. Read `docs/development-plan.md` and take the first prompt whose status is `not started`.
2. Read that prompt and the documents it names.
3. Implement only that prompt. Its `TestAcceptance_*` tests are the definition of done.
4. Update diagrams and prose when the behavior they describe changes.
5. Set that prompt’s row to `done`.
6. Stop.

Patterns you may use are listed in `docs/patterns.md`. A pattern that is not listed needs an ADR and a citation before any code uses it.
