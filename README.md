# hdsort

hdsort scans a folder on your machine, finds duplicate files, and asks you before it deletes anything. Files you keep are recorded in a local library. The next scan compares the new folder both with itself and with that library.

The review UI is a native window. From that window the owner can dry-run the current decisions and see what would change; a dry-run writes nothing. The files never leave the computer. Cloud copies are included only when a sync client has already stored them in a local folder.

This repository is the specification and the build plan. The module, package tree, version command, domain model, content-identity scan, keeper policy, SQLite library, apply, similar groups, exact-only scan mode, scan start, previews, the native review window, and the release workflow exist. The loopback review adapter remains tested and is not what `./hdsort` opens. Each file in [`prompts/`](prompts/) is one implementation session, in the order listed in [`docs/development-plan.md`](docs/development-plan.md).

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

`./hdsort` opens the native review window. It does not listen and it does not open a browser. The window keeps one list, grouped by folder, and opens the selected row as columns. Colors follow the system when it provides them, and otherwise the light or dark fallback. A build from source keeps the version `v0.0.0`. A release build replaces it with `-ldflags "-X main.version=vX.Y.Z"`.

## Install a release

Release tags follow [Semantic Versioning 2.0.0](https://semver.org/): `vMAJOR.MINOR.PATCH`, with an optional pre-release suffix `-alpha.N`, `-beta.N`, or `-rc.N`. Pushing that tag is the only version step. GitHub Actions builds each archive with cgo on a runner of that operating system, writes the tag into the binary, and attaches the archives to the [GitHub Release](https://github.com/cionz0/hdsort/releases). A local checkout does not change `v0.0.0`, and the next number is the next tag you push.

Download the archive for the machine and open it. Go is not required. On macOS, open `hdsort.app`: Finder shows its icon. On Windows, `hdsort_windows_amd64.exe` carries the same icon. On Linux, the archive also contains `hdsort.png` and `hdsort.desktop`; the file manager shows that icon after the desktop entry is installed.

| Machine | Archive |
| --- | --- |
| macOS Intel | `hdsort_darwin_amd64.tar.gz` (contains `hdsort.app`) |
| macOS Apple silicon | `hdsort_darwin_arm64.tar.gz` (contains `hdsort.app`) |
| Linux x86-64 | `hdsort_linux_amd64.tar.gz` |
| Linux arm64 | `hdsort_linux_arm64.tar.gz` |
| Windows x86-64 | `hdsort_windows_amd64.zip` (contains `hdsort_windows_amd64.exe`) |

The binary talks only to folders on that computer. It is not a cloud service.

## Status

Prompts 01 through 13 are done.

## License

Noncommercial use is free under the [PolyForm Noncommercial License 1.0.0](https://polyformproject.org/licenses/noncommercial/1.0.0). Every copy must keep this line:

```text
Required Notice: Copyright 2026 cionzo (https://github.com/cionz0/hdsort)
```

That notice is the credit.

Commercial use is not granted by that license. It needs a separate paid license from the copyright holder. Write to cionzoh@gmail.com. This repository does not set the price. A company using hdsort for its own files is commercial use. Personal, hobby, and noncommercial-organization use is free.
