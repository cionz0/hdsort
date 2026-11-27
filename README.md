# hdsort

hdsort scans a folder on your machine, finds duplicate files, and asks you before it deletes anything. Files you keep are recorded in a local library. The next scan compares the new folder both with itself and with that library.

The review UI is a page served by the program on `127.0.0.1`. From that page the owner can dry-run the current decisions and see what would change; a dry-run writes nothing. The files never leave the computer. Cloud copies are included only when a sync client has already stored them in a local folder.

This repository is the specification and the build plan. The module, package tree, version command, domain model, content-identity scan, keeper policy, SQLite library, apply, similar groups, exact-only scan mode, scan start, previews, the loopback review page, and the release workflow exist. Each file in [`prompts/`](prompts/) is one implementation session, in the order listed in [`docs/development-plan.md`](docs/development-plan.md).

## Documentation

| Document | What it settles |
| --- | --- |
| [docs/architecture.md](docs/architecture.md) | Context, containers, components, and the rules the code has to follow. Prose plus diagrams. |
| [docs/use-cases.md](docs/use-cases.md) | What the file owner can do, each with a sequence diagram. |
| [docs/patterns.md](docs/patterns.md) | The only design patterns, architectural styles, and practices the project may use, each with a citation. |
| [docs/adr/](docs/adr/) | Architecture decision records. |
| [docs/development-plan.md](docs/development-plan.md) | The prompt order and which prompts are done. |
| [prompts/](prompts/) | The session prompts. Acceptance tests inside each prompt are the definition of done. |

Project rules for contributors and agents live in [`.cursor/rules/`](.cursor/rules/): English artifacts, tests as acceptance, cited patterns only, and diagrams kept next to the prose.

## Build and run

Toolchain: Go 1.27 (stable 1.27.4 at the time of writing): <https://go.dev/dl/>.

```bash
go test ./...
go vet ./...
go build -o hdsort ./cmd/hdsort
./hdsort
```

`./hdsort` listens on `127.0.0.1` with an ephemeral port and prints the capability URL, `http://127.0.0.1:<port>/?token=<secret>`. It then opens that URL in the system browser. A build from source keeps the version `v0.0.0`. A release build replaces it with `-ldflags "-X main.version=vX.Y.Z"`.

## Install a release

Release tags follow [Semantic Versioning 2.0.0](https://semver.org/): `vMAJOR.MINOR.PATCH`, with an optional pre-release suffix `-alpha.N`, `-beta.N`, or `-rc.N`. Pushing that tag is the only version step. GitHub Actions writes the tag into the binary and attaches one archive per target to the [GitHub Release](https://github.com/cionz0/hdsort/releases). A local checkout does not change `v0.0.0`, and the next number is the next tag you push.

Download the archive for the machine, unpack it, and run the binary. Go is not required.

| Machine | Archive |
| --- | --- |
| macOS Intel | `hdsort_darwin_amd64.tar.gz` |
| macOS Apple silicon | `hdsort_darwin_arm64.tar.gz` |
| Linux x86-64 | `hdsort_linux_amd64.tar.gz` |
| Linux arm64 | `hdsort_linux_arm64.tar.gz` |
| Windows x86-64 | `hdsort_windows_amd64.zip` (contains `hdsort_windows_amd64.exe`) |

The binary talks only to folders on that computer. It is not a cloud service.

## Status

Prompts 01 through 11 are done.

## License

Not chosen yet.
