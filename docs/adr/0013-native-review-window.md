# ADR 0013: Native review window

- Status: Accepted
- Date: 2026-11-27

## Context

The review step is a progress bar, the file being read, a queue, the files already processed with their sizes, keeper suggestions, and image previews. A browser page on loopback could show that page, and ADR-0002 chose it so one Linux job could cross-compile every archive. The scan then returned only when it had finished, so the page waited on a spinner.

ADR-0012 builds each archive with cgo on the operating system it targets. That removes the reason to keep the browser as the product UI.

## Decision

`cmd/hdsort` opens a native window and does not listen, print a capability URL, or launch a browser. The window is drawn with Gio, an immediate-mode GUI for Go (Elias Naur and the Gio authors, 2019, <https://gioui.org>). The module is `gioui.org` v0.10.3.

`internal/adapters/gui` is the only package that imports Gio. Progress events and the view model live in `internal/app`. `internal/domain` imports only the standard library. Walking, hashing, SQL, and Trash run off the goroutine that paints the window. Tests do not open a display. They replace the window runner, the same kind of boundary fake as the folder dialog.

The HTTP adapter and its tests stay in the tree. They are no longer how the owner reviews a scan.

These parts of ADR-0002 stay: one binary per OS/architecture, no cloud API, release tags follow Semantic Versioning 2.0.0. Cgo stays enabled on the target OS (ADR-0012). The macOS binary stays unsigned.

## Consequences

- This supersedes ADR-0002. The product UI is the native window.
- A release still publishes the five archives named in ADR-0012. Their names do not change.
- Gio needs a C compiler on macOS and Linux. The Windows build links the window without turning cgo off.
- A system color is used when the operating system provides it. The light or dark fallback supplies every role the system does not. The window does not ship Gio’s indigo palette or a brand green. Linux reads the desktop color-scheme and does not link GTK.
- The list shows duplicate sets only. Paths on screen are relative to the chosen folder, and sizes use B, KB, MB, or GB. Each copy is a bordered column. A preview uses the operating-system thumbnail when that call returns one, and otherwise the JPEG, PNG, or GIF thumbnail. Linux does not link a toolkit to invent a thumbnail, so a non-image there has an empty preview well.
