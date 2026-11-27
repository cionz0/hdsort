# Prompt 01 — Skeleton

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 02.

## Read first

- `README.md`
- `docs/architecture.md` (package table)
- `docs/patterns.md` (Go module layout, `go test`, `go vet`)
- `docs/adr/0002-loopback-web-ui-in-one-go-binary.md`
- `docs/adr/0006-acceptance-tests-are-the-specification.md`
- `.cursor/rules/`

## Outcome

A Go module that builds a version command, with the package tree from `docs/architecture.md` present and a GitHub Actions workflow that runs the tests.

## In scope

- `go.mod` module path `github.com/cionz0/hdsort`, language version `1.27`. Toolchain note: current stable is Go 1.27.4 (`https://go.dev/dl/`).
- `cmd/hdsort` prints `hdsort v0.0.0` and exits 0. The version string is injected with `-ldflags` and defaults to `v0.0.0` when unset.
- Each internal package listed in the architecture package table exists and has a `doc.go` whose comment says which cited pattern it belongs to. No business behavior yet.
- `.github/workflows/test.yml` runs `go test ./...` and `go vet ./...` on Go 1.27.4 for `ubuntu-latest`, `macos-latest`, and `windows-latest`.
- Set prompt 01 to `done` in `docs/development-plan.md`.

## Out of scope

Domain types, hashing, HTTP, SQLite, and any third-party linter. `CGO_ENABLED` stays at its default in this workflow; prompt 11 sets `CGO_ENABLED=0` for releases.

## Acceptance tests

These tests are the definition of done.

- `TestAcceptance_VersionDefault` runs the version helper and gets `v0.0.0`.
- `go test ./...` and `go vet ./...` pass.
- The workflow file contains the three OS names and Go `1.27.4`.

## Patterns

Composition root in `cmd/hdsort`. Internal packages per the Go module layout. Do not add a pattern that is not in `docs/patterns.md`.
