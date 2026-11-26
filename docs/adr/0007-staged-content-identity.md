# ADR 0007: Staged content identity

- Status: Accepted
- Date: 2026-11-26

## Context

Hashing every byte of a large disk is the expensive part of a scan. Two files of different sizes cannot be identical. Files that share a size often still differ in the first or last block. The identity used for “exact duplicate” has to be the full contents, or a partial collision will merge unrelated files.

## Decision

Content identity is the BLAKE3 digest of the entire file, as specified by O’Connor, Aumasson, Neves, and Wilcox-O’Hearn (2020).

The hasher reaches that digest through three filters (Pipes and Filters):

1. Size. Different sizes never share a candidate set.
2. Partial hash. For a file of at least 128 KiB, BLAKE3 of the first 64 KiB and BLAKE3 of the last 64 KiB. Files in the same size bucket with different partial hashes are not fully read.
3. Full BLAKE3 of the bytes. This digest is the content ID.

Files smaller than 128 KiB skip step 2 and are hashed in full. Empty files are valid and share a content ID.

The partial hashes are a filter only. They are not stored as the content ID.

The Go module that implements BLAKE3 is chosen in prompt 03, must state that it implements the BLAKE3 specification, and is recorded in this ADR’s consequences when it is added. This repository does not reimplement the compression function.

## Consequences

- Two files with the same size and the same head and tail but a different middle produce two content IDs. That case is an acceptance test.
- A cloud placeholder or other dataless file is not hashed. The walk reports it as skipped.
- Symlinks are not followed. Each symlink is listed on the session as skipped, with reason `symlink` and the link destination.
- The hasher imports `lukechampine.com/blake3` v1.5.3. Its package comment states: "Package blake3 implements the BLAKE3 cryptographic hash function." The module README links that claim to the BLAKE3 specification repository, <https://github.com/BLAKE3-team/BLAKE3>. This repository calls that module and does not reimplement the compression function.
