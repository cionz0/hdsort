# Prompt 13 — Review by folder

Implement this prompt only. Stop when its acceptance tests pass. Do not start a later prompt.

## Read first

- `docs/architecture.md` (native review window)
- `docs/use-cases.md` (UC-1 through UC-4 and UC-7)
- `docs/adr/0010-similarity-algorithms.md`
- `docs/adr/0013-native-review-window.md`
- `prompts/12-native-window.md` (the window this prompt rearranges)
- `docs/patterns.md` (Ports and Adapters, Immediate-mode GUI)

## Outcome

The window has one file list, grouped by folder, and a compare pane for the selected row. A file’s status changes on that row. Copies of the selected row are columns, each with its preview. The owner can skip a whole set or one copy, and can keep every file of the same type in one folder while deleting copies that live in other folders. The progress bar sits with Start scan, Dry run, and Apply, and it names the action that is running. Each color is the system color when the operating system provides it, and otherwise the matching role from the light or dark fallback palette.

## In scope

- Replace the separate queue and processed lists with one list. A path appears once. Listing, queued, and reading update that row’s status. When the path joins an exact or similar group, the row becomes that group. The group is filed under the directory of the suggested keeper. A group with no recommendation is filed under the directory of its first path. Directories are ordered, and the rows of one directory stay together, so the owner can move through that directory before the next one.
- The selected row opens beside the list. Each member is a column: preview, path, size, and Keep, Delete, and Skip. JPEG, PNG, and GIF still use the existing thumbnail rules. Other types have no preview. The keeper text, the tie text, the sizes, and the synced-folder banner stay as they are.
- Skip on a column records `skip` for that path. Skip on the set records `skip` for every member. Both go through `domain.RecordDecision`.
- An exact row offers “Apply to all files of this type in <folder>”, where the folder is the directory the row is filed under. The type is exact image when the open file is JPEG, PNG, or GIF, and exact file otherwise. Applying it records `keep` for every member of every matching exact group that lives in that folder, and `delete` for members of those groups that live in another folder. A group with no member in that folder is left alone. A similar group is never a target and still has no delete suggestion. Two copies in the chosen folder are both kept.
- Start scan, the progress bar, Dry run, and Apply share one row. The bar’s label is the current action: listing the folder, reading a path with its hasher stage, or trashing a path. Only the control for that action is disabled, and the label is the reason. Listing still uses an indeterminate bar until the total is known.
- Ship two fallback palettes. Light is background `#F5F5F7`, text `#1C1C1E`, primary `#1C1C1E`, text on primary `#FFFFFF`. Dark is background `#1C1C1E`, text `#F5F5F7`, primary `#F5F5F7`, text on primary `#1C1C1E`. Read light or dark on macOS, Windows, and Linux, and repaint when it changes. That choice selects the fallback. For each role, use the system color when the operating system provides that role, and the fallback role otherwise. A missing accent does not change the background. Do not link GTK. Do not use `material.NewTheme`’s indigo `#3F51B5`, and do not use a brand green.
- Add one consequence to ADR-0013: a system color is used when the operating system provides it, and the light or dark fallback supplies every role the system does not. Do not add a pattern.

Update `docs/architecture.md` and `docs/use-cases.md` in the same change so the prose and the diagrams describe this one list, the compare columns, and the system appearance.

## Out of scope

A new keeper rank. A delete suggestion for similar files. A new similarity algorithm. Deleting the HTTP package. Code signing. Linking GTK to copy a desktop theme. Gio’s default indigo palette. A brand green.

## Acceptance tests

Drive the view model. Use a temporary directory. Do not open a window and do not require a display. Pass the appearance and the system colors in; do not read the developer’s operating system.

- `TestAcceptance_ReviewListsOneRowPerSetUnderItsFolder`: an exact image with two copies in one folder and one copy in another folder is one row, filed under the suggested keeper’s folder. The path is not also listed as queued or processed.
- `TestAcceptance_ScanStatusStaysOnTheSameRow`: a queued path that later reports a hasher stage updates that row. The model does not grow a second list.
- `TestAcceptance_FolderOrderWalksOneDirectory`: two groups whose keeper is in the same folder come before a group whose keeper is in another folder.
- `TestAcceptance_SkipSetSkipsEveryMember`: skipping the set records `skip` for every member and records no `delete`.
- `TestAcceptance_ApplyFolderKeepsThatFolder`: applying the folder action to exact images keeps every image member in that folder, deletes the image member in another folder, and does not record a decision for an exact text group or a similar group.
- `TestAcceptance_RunningActionDisablesItsControl`: a reading event names the path and the stage, and disables only Start scan. A trash event names the path and disables only Apply.
- `TestAcceptance_SystemPaletteFollowsAppearance`: with no system colors, light appearance is the light fallback and dark appearance is the dark fallback, including text on the primary control. A system background with no accent keeps that background and takes the primary color from the fallback for that appearance. A full set of system colors is used as given and does not keep a fallback role. The primary color is never `#3F51B5`.

`go test ./...` and `go vet ./...` pass with cgo enabled.

## Patterns

Gio stays the immediate-mode shell in `internal/adapters/gui`. The view model and the appearance-to-palette mapping live in `internal/app`, which does not import Gio. Reading the operating system appearance is the same kind of boundary as the folder dialog.
