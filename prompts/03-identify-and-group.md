# Prompt 03 — Identify and group

Implement this prompt only. Stop when its acceptance tests pass. Do not start prompt 04.

## Read first

- `docs/architecture.md` (content identity, walk rules)
- `docs/use-cases.md` (UC-1)
- `docs/adr/0007-staged-content-identity.md`
- `docs/patterns.md` (Pipes and Filters, BLAKE3, Ports and Adapters)

## Outcome

A scan of a fixture directory produces content IDs and exact groups. Files that are not duplicates do not appear in a group.

## In scope

- A hasher port declared in `internal/app`, implemented in `internal/adapters/hash`.
- Choose a Go module that states it implements the BLAKE3 specification. Record the module path and version in the consequences of ADR-0007. Do not reimplement BLAKE3.
- Filters: size, then head/tail 64 KiB when the file is at least 128 KiB, then full BLAKE3. The stored content ID is the full digest only.
- A walk port: do not follow symlinks; skip directory names `.git` and `node_modules`; skip a data-directory path passed into the walk; represent a dataless file as skipped. The dataless bit is a field the walk adapter sets. Tests set that field through a fixture file info or a temp file the adapter is taught to treat as dataless via an explicit test hook that is not a production shortcut. Prefer a small `Availability` value the adapter returns, and a test double only for the platform flag that temp files cannot raise. The rest of the walk uses a real temporary directory.
- Exact grouping of the readable files. Unique content IDs produce no group.

## Out of scope

Keeper recommendations, the library, similar files, HTTP.

## Acceptance tests

Use a temporary directory.

- `TestAcceptance_IdenticalBytesShareContentID`, including two empty files.
- `TestAcceptance_DifferentMiddleDifferentContentID`: same size, same first 64 KiB, same last 64 KiB, different middle, two content IDs. Build the files larger than 128 KiB.
- `TestAcceptance_SmallFilesSkipPartialFilter`: two different files under 128 KiB still get distinct full-hash IDs.
- `TestAcceptance_ExactGroupOfThree`: three copies form one group; a unique file is absent from every group.
- `TestAcceptance_SymlinkNotFollowed`: a symlink to a file inside the tree is not a group member.
- `TestAcceptance_SkipGitAndNodeModules`: files under those directory names are not hashed.
- `TestAcceptance_DatalessFileSkipped`: a file reported as dataless is listed on the scan result as skipped and has no content ID.
- `TestAcceptance_PartialHashDoesNotFullReadDifferentHeads`: when two large files differ in the first 64 KiB, the hasher does not read past the partial window. Instrument the reader port with a counting reader in the test.

Update the UC-1 diagram only if the scan result fields you added are not already described.

## Patterns

Pipes and Filters for the three hash stages. The walk and the hasher are driven adapters. The grouping function is a pure domain function.
