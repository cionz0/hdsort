# hdsort

hdsort scans a folder on your machine, finds duplicate files, and asks you before it deletes anything. Files you keep are recorded in a local library. The next scan compares the new folder both with itself and with that library.

The review UI is a page served by the program on `127.0.0.1`. The files never leave the computer. Cloud copies are included only when a sync client has already stored them in a local folder.

This repository is the specification and the build plan. The program itself is not implemented yet. Each file in [`prompts/`](prompts/) is one implementation session, in the order listed in [`docs/development-plan.md`](docs/development-plan.md).

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

Not yet. Prompt 01 adds the Go module. Prompt 10 adds Release binaries for macOS, Linux, and Windows.

When the module exists, the toolchain is Go 1.27 (stable 1.27.4 at the time of writing): <https://go.dev/dl/>.

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 go build -o hdsort ./cmd/hdsort
```

`go` is required on a developer machine. A Release binary does not require a Go installation.

## Status

Specification only. Next session: [`prompts/01-skeleton.md`](prompts/01-skeleton.md).

## License

Not chosen yet.
