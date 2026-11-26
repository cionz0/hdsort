# ADR 0006: Acceptance tests are the specification

- Status: Accepted
- Date: 2026-11-26

## Context

The work is split into prompts so that each session has a finish line. Reviewing a prompt by reading the code alone does not tell a later session whether the product behavior held.

## Decision

The acceptance tests named in each prompt are the specification of that slice. They are Go tests named `TestAcceptance_<Behavior>`.

A prompt is done when those tests pass, `go test ./...` passes, `go vet ./...` passes, and any doc the behavior changed (including its diagram) is updated in the same change.

A port fake is allowed for the OS folder dialog, the macOS/Windows Trash UI, and the platform flag that marks a cloud placeholder as dataless (a temporary file cannot raise that flag). Filesystem walks, hashing, SQLite, HTTP, and the Freedesktop trash layout are tested with real temp directories, real SQLite files, and `httptest`.

## Consequences

- Prompts list tests before implementation notes.
- A refactor that keeps the same tests needs no new ADR. A behavior change adds or edits a `TestAcceptance_*` test and the doc that describes it.
