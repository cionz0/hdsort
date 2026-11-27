# ADR 0011: Exact-only is a scan mode

- Status: Accepted
- Date: 2026-11-27

## Context

ADR-0010 runs the similarity strategy after exact grouping on every scan. The owner also wants a switch that limits a scan to byte-identical files. Two files that are only near-duplicates must then stay ungrouped.

The review page in prompt 08 shows whatever similar groups the session already has. It does not start the scan. The switch belongs to the action that starts the scan.

## Decision

A scan has a mode. The default mode runs similarity exactly as ADR-0010 describes. The exact-only mode does not call the similarity strategy. Similar groups on that session are empty. Exact groups are unchanged: byte-identical files are still one exact group.

The mode is an explicit argument of the scan use case. A missing or nil similarity strategy is not the switch. The strategy remains a required dependency, and exact-only decides whether the use case calls it.

The mode is stored on the session. The default, including an omitted value, is to include similar files.

`POST /scans` accepts `exactOnly`. Prompt 10 places the control on the action that starts the scan and passes that field through. This ADR does not change the average-hash or simhash thresholds.

## Consequences

- Prompt 09 implements the mode, the session field, and the acceptance tests. It does not add `POST /scans`.
- Prompt 10 reads `exactOnly` and does not redefine similarity.
- Existing scans that omit the flag keep today’s similar groups. The prompt 07 tests stay valid.
