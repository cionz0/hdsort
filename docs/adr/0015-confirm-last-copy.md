# ADR 0015: Confirm before trashing the last copy

- Status: Accepted
- Date: 2026-11-28

## Context

ADR-0004 rejects a plan that would trash every known copy of a content ID the library does not keep. The window showed that rejection as the content ID. The owner could not tell which files were involved, and could not choose to trash them anyway.

## Decision

Apply and Dry run still detect that plan and still write nothing until the owner confirms. The window asks with one line per file. The line is the path relative to the scan root. The content ID is not shown. An accent button, “Trash the last copies”, runs the plan.

Without that confirmation the plan is still rejected. With it, Apply trashes every copy, does not upsert that content ID, and marks the session applied. A trash path with no delete decision is still rejected.

## Consequences

- ADR-0004 stays Accepted. Trash still runs before any library write, and a failed trash still leaves the library unchanged.
- The owner can remove the last copy of a file that the library does not already keep, after seeing the paths.
