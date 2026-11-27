# ADR 0012: Release builds enable cgo on the target operating system

- Status: Accepted
- Date: 2026-11-27

## Context

A native window on macOS and Linux links system libraries (AppKit, and the GL loader Gio uses). Those bindings are cgo. `CGO_ENABLED=0` cannot compile them, and a Linux runner cannot link a macOS binary once cgo is on.

The repository is public. Standard GitHub-hosted runners for a public repository, including macOS and Windows, do not draw from the account's included compute usage.

ADR-0002 required `CGO_ENABLED=0` so one Linux job could cross-compile every archive. That rule blocks the window.

## Decision

Release builds set `CGO_ENABLED=1`. Each archive is built on a runner of its operating system:

| Archive | Runner |
| --- | --- |
| `darwin/amd64`, `darwin/arm64` | `macos-latest` |
| `linux/amd64` | `ubuntu-latest` |
| `linux/arm64` | `ubuntu-24.04-arm` |
| `windows/amd64` | `windows-latest` |

`darwin/amd64` is built on the Apple silicon runner with `clang -arch x86_64`. Archive names stay the ones ADR-0002 and the icon release already publish. Tags stay Semantic Versioning 2.0.0. One binary per OS/architecture. No cloud API.

SQLite stays `modernc.org/sqlite`. Cgo is for the window, not for the database.

## Consequences

- This supersedes the `CGO_ENABLED=0` sentence in ADR-0002. The loopback review page remains the product UI until the native window lands.
- A release tag runs five build jobs, then one job publishes the archives. A job that fails does not cancel the others.
- The macOS binary is still unsigned. Gatekeeper still asks the owner to approve it. Signing stays out of scope.
- Tests run on Ubuntu, macOS, and Windows. The host C compiler is enough; the test job does not cross-compile.
