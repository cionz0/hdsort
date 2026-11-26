# Prompt 10 — Release binaries

Implement this prompt only. Stop when its acceptance tests pass.

## Read first

- `docs/adr/0002-loopback-web-ui-in-one-go-binary.md`
- `docs/patterns.md` (Semantic Versioning, `CGO_ENABLED=0` implied by the one-binary decision)
- `README.md`

## Outcome

A tag `vMAJOR.MINOR.PATCH` builds one archive per target and attaches it to a GitHub Release. The README tells a person how to download the archive and how to build from source.

## In scope

- `.github/workflows/release.yml`, triggered on tags that match `v[0-9]+.[0-9]+.[0-9]+` and the same with a SemVer pre-release suffix (`-alpha.1`, `-beta.1`, `-rc.1`). Reject tags that are missing a segment, such as `v1.2`.
- `CGO_ENABLED=0` for every target: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`.
- `-ldflags` sets the version printed by prompt 01 to the tag.
- Artifact names: `hdsort_<os>_<arch>` plus `.exe` on Windows, packed as `.tar.gz` or `.zip`.
- README sections: what the program is (already there), how to build (`go build` with Go 1.27), how to run, and how to install a Release. Keep the “not a cloud service” facts accurate.
- Set prompt 10 to `done` in `docs/development-plan.md` only after the workflow file is in the tree and the local build below passes. You do not need to push a tag in this session.

## Out of scope

Code signing, notarization, Homebrew, and cloud API clients.

## Acceptance tests

- `TestAcceptance_VersionFromLdflags` is already covered if prompt 01 reads the ldflag variable. Add the test now if it is missing: the helper returns the value it was given.
- A small test or a `go test` in `cmd/hdsort` is not required to invoke GitHub. Instead, document in the workflow file and add `TestAcceptance_ReleaseTargets` only if you keep the target list in Go. If the target list lives solely in YAML, the done check is: `CGO_ENABLED=0 go build -o /tmp/hdsort ./cmd/hdsort` succeeds on the host, and the workflow YAML contains each of the five OS/arch pairs.
- Do not weaken `go test ./...`.

## Patterns

No new pattern. The release workflow is the shell around the same `cmd/hdsort` composition root.
