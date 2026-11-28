# Prompt 19 — Folder outline

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-1, UC-2), and `prompts/14-readable-review.md`.

Prompt 14 is already in effect: the list shows duplicate sets only, and a path drawn in the window is relative to the scan root. Prompt 18 is already in effect: selecting a set still opens the compare columns and the radios.

## Outcome

The left pane is an expandable outline of folders, in the same sense as Finder’s list view. Each folder shows only its own name, indented by its depth, with a disclosure control. Expanding a folder shows the folders and the duplicate sets inside it. Collapsing it hides those descendants.

A set stays filed under the same folder as today: the suggested keeper’s directory, or the first member’s directory when the group has no recommendation. Every directory from the scan root down to that folder is a node, including a directory that has no set of its own. A directory that is not on that path is absent. A file that is unique, a symlink, or a dataless placeholder still adds no row and no directory.

The scan root is the top node. Its name is the base name of the chosen folder. On a new scan the root starts expanded and every other directory starts collapsed. Expansion is window state for the current scan. It is not a decision, and a new scan resets it. Collapsing a directory does not change the selected set. The compare pane keeps showing that set when its row is hidden.

A folder outside the scan root is one node. Its name is that folder’s cleaned absolute path. It is not split into parents above the root.

Siblings are ordered by name, using the same string order the list uses today. A directory sorts before a set when the names are equal. Build the paths with `filepath` so the tests do not depend on one slash character.

Clicking the disclosure toggles that directory. Clicking a set selects it. Clicking a directory does not select a set.

## Acceptance tests

Drive the outline from the folder rows and the set of expanded directories. Use a temporary directory. Do not open a window.

- `TestAcceptance_OutlineUsesOneNamePerLevel`: sets filed under `a/b` and `a/c`, with every directory expanded, show the root base name, then `a`, then `b` and its set, then `c` and its set. No node name contains a path separator. Depth increases by one at each directory.
- `TestAcceptance_CollapsedDirectoryHidesItsContents`: with only the root expanded, the visible nodes are the root and `a`. Expanding `a` and leaving `b` and `c` collapsed shows `b` and `c` and hides both sets.
- `TestAcceptance_IntermediateFoldersStayInTheOutline`: a set filed only under `a/b/c` still inserts `a` and `b` when those directories are expanded.
- `TestAcceptance_FolderOutsideTheRootStaysOneNode`: a set whose folder is outside the scan root is one node named with that absolute path. The outline does not invent a parent above the root.

`go test ./...` and `go vet ./...` pass with cgo enabled.

## Notes

Do not change which files are listed, the keeper rank, the similar-group delete rule, Open, the radios, the progress label, or the release archive names. Decisions still store the cleaned absolute path. Update the Review window prose and its sequence diagram, and the UC-1 and UC-2 notes, in the same change: the owner expands a folder, then selects a set.
