# Pattern catalog

This is the closed list of architectural styles, design patterns, and engineering practices hdsort may use. Each entry cites a published source. Adding an entry requires an ADR that says why this codebase needs it. See `.cursor/rules/established-patterns.mdc`.

The words in this file are the names from those sources. Packages may use the shorter Go name in the right-hand column.

## Architectural styles

| Name in this repo | Source | Where it applies |
| --- | --- | --- |
| Ports and Adapters (Hexagonal Architecture) | Alistair Cockburn, “Hexagonal Architecture,” 2005. <https://alistair.cockburn.us/hexagonal-architecture/> | The process is the application inside the hexagon. The native window, filesystem, SQLite, and Trash sit outside and talk through ports. |
| Dependency Rule | Robert C. Martin, “The Clean Architecture,” 2012. <https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html> | Source dependencies point toward `internal/domain`. Domain imports none of the other internal packages. Adapters are wired in `cmd/hdsort` only. |
| Functional core, imperative shell | Gary Bernhardt, “Boundaries,” 2012. <https://www.destroyallsoftware.com/talks/boundaries> | Content identity, grouping, keeper ranking, and plan validation are pure functions. Walking, hashing, SQL, the window, and Trash are the shell. |
| C4 model | Simon Brown, “The C4 model for visualising software architecture.” <https://c4model.com/> | `docs/architecture.md` uses System Context, Container, and Component. Mermaid is the rendering format, not a C4 notation. GitHub renders Mermaid in Markdown: <https://github.blog/2022-02-14-include-diagrams-markdown-files-mermaid/> |
| Architecture Decision Records | Michael Nygard, “Documenting Architecture Decisions,” 2011. <https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions> | `docs/adr/`. Status, context, decision, consequences. |

## Design patterns

| Name in this repo | Source | Where it applies |
| --- | --- | --- |
| Strategy | Erich Gamma, Richard Helm, Ralph Johnson, John Vlissides, *Design Patterns*, Addison-Wesley, 1994, “Strategy.” | Interchangeable keeper policy, similarity policy, and layout policy. The use case depends on the policy interface, not on a concrete rule. |
| Immediate-mode GUI | Elias Naur and the Gio authors, Gio, immediate-mode GUI for Go, 2019. <https://gioui.org> | `internal/adapters/gui`. The use cases do not import Gio. |
| Pipes and Filters | Frank Buschmann, Regine Meunier, Hans Rohnert, Peter Sommerlad, Michael Stal, *Pattern-Oriented Software Architecture, Volume 1*, Wiley, 1996, “Pipes and Filters.” The same staging is how `jdupes` avoids hashing every byte twice: <https://github.com/jbruchon/jdupes> | Size filter, then a partial-hash filter, then a full BLAKE3 filter. A later filter runs only for files the earlier filter could not separate. |
| Repository | Eric Evans, *Domain-Driven Design*, Addison-Wesley, 2003, “Repositories.” | `LibraryRepository` and `SessionRepository` hide SQLite. Use cases see collection-style operations, not SQL. |

## Go practices

These are language practices from the Go project, not additional architecture.

| Practice | Source | Where it applies |
| --- | --- | --- |
| Module layout: one module at the repo root, commands under `cmd/`, private packages under `internal/` | “Organizing a Go module,” <https://go.dev/doc/modules/layout>. Internal directories: <https://go.dev/doc/go1.4#internalpackages> | `cmd/hdsort`, `internal/domain`, `internal/app`, `internal/adapters/...` |
| Interfaces belong to the consumer | “Go Code Review Comments,” Interfaces. <https://go.dev/wiki/CodeReviewComments#interfaces>. Go Proverbs, “The bigger the interface, the weaker the abstraction.” <https://go-proverbs.github.io/> | Port interfaces are declared in `internal/app`, next to the use case that calls them. Adapters implement those interfaces. `cmd/hdsort` passes the concrete adapters in. |
| Accept interfaces, return concrete types | Go Proverbs. <https://go-proverbs.github.io/> | Constructors in adapters return the concrete type. Use-case functions accept the small port they call. |
| Table-driven tests | “TableDrivenTests,” Go Wiki. <https://go.dev/wiki/TableDrivenTests> | Acceptance cases that differ by input rows. |
| `go test` and `go vet` | The `go` command. <https://pkg.go.dev/cmd/go> | The test workflow. A third-party linter needs its own ADR before it is added. |
| `crypto/rand` for secrets, `crypto/subtle` for comparison | `crypto/rand`, `crypto/subtle` in the Go standard library. | The loopback capability token. |

## Specification and safety practices

| Practice | Source | Where it applies |
| --- | --- | --- |
| Acceptance tests are the specification | Gojko Adzic, *Specification by Example*, Manning, 2011. Steve Freeman and Nat Pryce, *Growing Object-Oriented Software, Guided by Tests*, Addison-Wesley, 2009 (the outer acceptance loop). | Tests named `TestAcceptance_<Behavior>`. The prompt files list them before the code exists. |
| Semantic Versioning for release tags | Tom Preston-Werner, Semantic Versioning 2.0.0. <https://semver.org/> | Git tags `vMAJOR.MINOR.PATCH`. Pre-release tags use a dot after the identifier (`v0.1.0-alpha.1`). |
| Loopback-only administrative UI | The HTTP server binds `127.0.0.1` or `::1`. W3C Secure Contexts treats `http://127.0.0.1` as a potentially trustworthy origin. <https://www.w3.org/TR/secure-contexts/#localhost> | Superseded for the product UI by ADR-0013. The HTTP adapter remains tested. |
| Opaque identifiers instead of raw paths | OWASP, “Path Traversal.” <https://owasp.org/www-community/attacks/Path_Traversal> | Preview and delete commands take a file id issued by the session. A request that supplies a filesystem path is rejected. |
| SQLite WAL mode | SQLite, “Write-Ahead Logging.” <https://www.sqlite.org/wal.html> | The library file. One writer: the hdsort process. |
| XDG Base Directory for the data file on Linux | Freedesktop, “XDG Base Directory Specification.” <https://specifications.freedesktop.org/basedir-spec/latest/> | `$XDG_DATA_HOME/hdsort` or `~/.local/share/hdsort`. |
| Application Support on macOS | Apple, “The Library Directory Stores App-Specific Files,” File System Programming Guide. <https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/FileSystemOverview/FileSystemOverview.html> | `~/Library/Application Support/hdsort`. |
| Roaming AppData on Windows | Microsoft, `FOLDERID_RoamingAppData`. <https://learn.microsoft.com/en-us/windows/win32/shell/knownfolderid> | `%AppData%\hdsort`. |
| Trash on Linux | Freedesktop, “Trash specification.” <https://specifications.freedesktop.org/trash-spec/latest/> | The Trash adapter. Files are moved to Trash, not unlinked. |
| BLAKE3 as the content identity | Jack O’Connor, Jean-Philippe Aumasson, Samuel Neves, Zooko Wilcox-O’Hearn, “BLAKE3,” 2020. <https://github.com/BLAKE3-team/BLAKE3-specs/blob/master/blake3.pdf> | The content ID is the BLAKE3 digest of the full file. The hasher imports `lukechampine.com/blake3`, whose package states that it implements the BLAKE3 hash function (ADR-0007). The hash is not reimplemented in this repo. |

## Explicitly not adopted

These appear in Go discussions and are easy to copy by habit. They are not in use here:

- `github.com/golang-standards/project-layout` is a community template, not the layout documented by the Go project. This repo follows <https://go.dev/doc/modules/layout> instead.
- A hosted website, a cloud-provider API, and an automatic delete are outside the decisions in `docs/adr/`.
