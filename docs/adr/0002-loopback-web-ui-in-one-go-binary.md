# ADR 0002: Loopback web UI in one Go binary

- Status: Accepted
- Date: 2026-11-26

## Context

The program has to run on macOS, Linux, and Windows from a GitHub Release, without a language runtime installed on the machine. The review step needs a visual page, especially once images are involved. The files stay on the machine that owns them.

A page opened from a public website cannot read that disk. A file input in the browser does not give the process a path it can walk.

## Decision

Ship one Go binary per OS/architecture. The binary is the process. It listens only on the loopback interface, opens the system browser, and serves the review UI from that process.

The user opens a capability URL `http://127.0.0.1:<port>/?token=<secret>`. The secret is at least 128 bits from `crypto/rand`. The first response sets an `HttpOnly`, `SameSite=Strict` cookie named `hdsort_token`. Later requests send that cookie or an `Authorization: Bearer` header. Comparison uses `crypto/subtle`.

Choosing a folder is a port implemented by a native folder dialog in the process. The page asks the process to open the dialog. Tests supply a fake dialog that returns a fixture path.

Release tags follow Semantic Versioning 2.0.0. The release build enables cgo and runs on the target operating system (ADR-0012).

## Consequences

- The UI is static HTML served by the binary, plus the JSON routes in `docs/architecture.md`. A Node toolchain is not required to build or run.
- `http://127.0.0.1` is a potentially trustworthy origin under the W3C Secure Contexts specification, so the cookie can be marked `Secure` on loopback.
- A binary downloaded on macOS still needs the user to pass Gatekeeper unless it is signed and notarized. Signing is a later release concern. Building from source is the path that avoids that prompt.
- Cloud provider APIs are out of scope. Cloud files are visible only when a sync client has already stored them on a local path.
