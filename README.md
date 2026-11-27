# hdsort

hdsort scans a folder on your machine, finds duplicate files, and asks you before it deletes anything. Files you keep are recorded in a local library. The next scan compares the new folder both with itself and with that library.

The review UI is a page served by the program on `127.0.0.1`. From that page the owner can dry-run the current decisions and see what would change; a dry-run writes nothing. The files never leave the computer. Cloud copies are included only when a sync client has already stored them in a local folder.

This repository is the specification and the build plan. The module, package tree, version command, domain model, content-identity scan, keeper policy, SQLite library, apply, similar groups, and the loopback review page exist. Each file in [`prompts/`](prompts/) is one implementation session, in the order listed in [`docs/development-plan.md`](docs/development-plan.md).

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

`./hdsort` listens on `127.0.0.1` with an ephemeral port and prints the capability URL, `http://127.0.0.1:<port>/?token=<secret>`. It then opens that URL in the system browser. The embedded version stays `v0.0.0` until a release build replaces it with `-ldflags "-X main.version=vX.Y.Z"`. Prompt 11 adds the Release binaries for macOS, Linux, and Windows. A Release binary does not require a Go installation.

## Status

Prompts 01 through 08 are done. Next session: [`prompts/09-exact-only.md`](prompts/09-exact-only.md).

## License

Not chosen yet.
