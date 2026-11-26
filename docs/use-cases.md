# Use cases

The person in these use cases is the owner of the files. The system is the hdsort process and the browser window it opens. Acceptance tests for each use case are listed in the prompt that implements it (`docs/development-plan.md`).

```mermaid
flowchart LR
  owner[File owner]
  uc1[UC-1 Scan a folder]
  uc6[UC-6 Reconcile the library]
  uc2[UC-2 Review exact duplicates]
  uc3[UC-3 Review similar files]
  uc5[UC-5 Review layout proposals]
  uc7[UC-7 Dry-run the plan]
  uc4[UC-4 Apply the confirmed plan]
  owner --> uc1
  uc1 --> uc6
  uc1 --> uc2
  uc1 --> uc3
  uc1 --> uc5
  uc2 --> uc7
  uc3 --> uc7
  uc5 --> uc7
  uc2 --> uc4
  uc3 --> uc4
  uc5 --> uc4
```

## UC-1 Scan a folder

The owner points hdsort at a folder. hdsort inventories the readable files, assigns content IDs, and opens a session.

The walk does not follow symlinks, does not hash dataless placeholders, and skips `.git`, `node_modules`, Trash, and the hdsort data directory. Skipped placeholders are listed on the session so the owner can see that those paths were not compared.

```mermaid
sequenceDiagram
  actor Owner
  participant Page as Review page
  participant Scan as Scan use case
  participant Disk as Filesystem
  participant Hash as Hasher
  Owner->>Page: Choose folder
  Page->>Scan: Start scan
  Scan->>Disk: Walk the root
  Disk-->>Scan: Readable files and skipped placeholders
  Scan->>Hash: Content ID for each readable file
  Hash-->>Scan: Content IDs
  Scan-->>Page: Open session
```

Two readable files with the same bytes produce the same content ID, including two empty files. Two files that share a size and share the first and last 64 KiB but differ in the middle produce different content IDs.

## UC-2 Review exact duplicates

The owner looks at exact groups. Each group is either given one recommendation or left without one when the keeper policy ties.

```mermaid
sequenceDiagram
  actor Owner
  participant Page as Review page
  participant Policy as Keeper policy
  Page->>Policy: Exact group
  alt A library survivor or a plain name wins
    Policy-->>Page: Keeper, delete list, reason code
  else Names and library rank still tie
    Policy-->>Page: No recommendation
  end
  Owner->>Page: Keep, delete, or skip each file
```

The recommendation the owner sees follows the rank in `docs/architecture.md`:

- A library survivor outside the folder is the keeper. The in-folder copies are the delete suggestion. The library file is not given a new path.
- Among copies that are all inside the folder, a name that is not a generated copy outranks `name (1)`, `Copy of name`, and a Dropbox `conflicted copy` name.
- When those signals tie, the newer modification time wins, and the reason code is `newer_mtime_tiebreak`.
- When the modification times also tie, there is no recommendation.

`report_v1.pdf` and `report_final.pdf` with different bytes do not form a group. With identical bytes they do, and the version words still do not pick the keeper.

The owner’s decision is stored against the file id. Nothing is trashed in this use case.

When the scan root has a synced-folder path segment (`CloudStorage`, `Mobile Documents`, `Dropbox`, or `Google Drive`), the page shows the banner from `docs/architecture.md` before the owner confirms a delete.

## UC-3 Review similar files

The owner inspects files that are close but not byte-identical. The page shows them apart from exact groups and does not suggest which one to delete.

```mermaid
sequenceDiagram
  actor Owner
  participant Page as Review page
  participant Sim as Similarity strategy
  Sim-->>Page: Similar group, no delete recommendation
  Owner->>Page: Explicit keep, delete, or skip
  Note over Page: A byte-identical pair never arrives here
```

The strategy itself is chosen later, in an ADR written at the start of prompt 07, because the algorithm has to be a cited one. The behavior this use case already fixes is the separation: no recommended deletes, and no overlap with exact groups.

## UC-4 Apply the confirmed plan

The owner asks hdsort to carry out the decisions. hdsort checks the plan, moves confirmed deletes to Trash, and only then records survivors.

```mermaid
sequenceDiagram
  actor Owner
  participant Apply as Apply use case
  participant Trash as Trash port
  participant Library as Library repository
  Owner->>Apply: Apply session
  Apply->>Apply: Reject unconfirmed deletes and last-copy loss
  Apply->>Trash: Move each confirmed delete
  alt Every trash call succeeded
    Trash-->>Apply: Ok
    Apply->>Library: Upsert survivors
    Apply-->>Owner: Session applied
  else A trash call failed
    Trash-->>Apply: Error
    Apply-->>Owner: Session still open, library unchanged
  end
```

The last-copy rule: if the content ID is not already a valid library entry, the plan must leave at least one copy outside Trash. The owner can delete in-folder copies of a file the library already keeps.

A synced-folder banner does not block apply. It is shown in UC-2 so the confirmation happens with that warning already visible.

## UC-5 Review layout proposals

The owner sees a list of moves for the files that remain. The default strategy’s list is empty, so a folder that only needed deduplication shows no moves.

```mermaid
sequenceDiagram
  actor Owner
  participant Layout as Layout strategy
  participant Page as Review page
  participant Disk as Filesystem
  Layout-->>Page: Zero or more moves inside the scan root
  alt The owner confirms a move
    Owner->>Page: Confirm move
    Page->>Disk: Move inside the root, on apply
  else The list is empty or the owner skips
    Page-->>Owner: Paths stay where they are
  end
```

A proposal whose destination leaves the scan root is rejected and is not shown as a confirmable move. Confirming a move does not run it until UC-4 apply. After a successful apply, the library path is the destination path.

## UC-6 Reconcile the library

UC-1 runs this before it compares the folder with the library. Entries that point outside the folder are left alone. Entries that point inside the folder are checked against the disk.

```mermaid
sequenceDiagram
  participant Scan as Scan use case
  participant Library as Library repository
  participant Disk as Filesystem
  Scan->>Library: Entries under the scan root
  loop Each entry
    Scan->>Disk: Still at the stored path, same content ID?
    alt Missing or content changed
      Disk-->>Scan: No
      Scan->>Library: Mark stale
    else Unchanged
      Disk-->>Scan: Yes
      Scan->>Library: Refresh metadata
    end
  end
```

A stale entry does not make the remaining files look like duplicates of a missing original. Two remaining copies still become an exact group by their own content IDs. One remaining copy is a survivor candidate, not a duplicate group.

## UC-7 Dry-run the plan

The owner asks the review page what the current decisions would do. hdsort runs the same checks as UC-4 and shows the Trash paths, the confirmed moves, and the library rows it would write. Files, Trash, and the library stay as they were. The session stays `open`. Apply remains a separate action.

```mermaid
sequenceDiagram
  actor Owner
  participant Page as Review page
  participant Plan as Plan preview
  participant Disk as Filesystem and library
  Owner->>Page: Dry run
  Page->>Plan: Same checks as apply
  alt The plan is valid
    Plan-->>Page: Trash list, moves, library upserts
    Page-->>Owner: Report, and Nothing was changed
    Note over Disk: No writes
  else The plan is rejected
    Plan-->>Page: Same rejection apply would return
    Page-->>Owner: Rejection, and Nothing was changed
    Note over Disk: No writes
  end
```

A dry-run does not record a new decision and does not replace Apply.

## Out of scope for these use cases

- Calling a cloud provider API, or hydrating a dataless file so it can be hashed.
- Deleting without a decision recorded on that file id.
- Proposing a folder layout that moves files out of the scan root.
- Choosing a keeper from a similar group on the owner’s behalf.
