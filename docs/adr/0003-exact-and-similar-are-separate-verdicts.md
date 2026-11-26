# ADR 0003: Exact and similar are separate verdicts

- Status: Accepted
- Date: 2026-11-26

## Context

People keep more than one kind of “duplicate”: byte-identical copies, renamed copies, and files that merely look alike (a recompressed photo, a revised document). Treating the third kind as permission to delete loses originals.

## Decision

A scan produces two independent results.

- An **exact group** contains two or more files with the same content ID (ADR-0007). The keeper policy may attach one delete recommendation, with a reason code from the closed list in `docs/architecture.md`.
- A **similar group** contains files that a cited similarity strategy considers close and that are not byte-identical. It carries no delete recommendation. The user can still mark a member for deletion by an explicit decision.

Byte-identical files are exact only. They are not also placed in a similar group.

Version-looking names (`report_v1`, `report_final`) do not create a group and do not choose a keeper. They matter only when the content IDs are already equal, and then only through the generated-copy naming rules or the modification-time tie-break.

## Consequences

- The review UI renders the two results in separate sections.
- A similarity algorithm is added only after an ADR cites the algorithm and states its threshold. Prompt 07 is the first place that ADR is allowed.
