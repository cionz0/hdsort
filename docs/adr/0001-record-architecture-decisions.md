# ADR 0001: Record architecture decisions

- Status: Accepted
- Date: 2026-11-26

## Context

hdsort is specified before it is implemented. Later sessions will each execute one prompt. Those sessions need a stable place to read why a choice was made, and a rule for how to change it.

## Decision

Significant design choices are recorded in this directory using the ADR format described by Michael Nygard (2011): context, decision, and consequences.

A change that reverses an ADR sets that ADR to Superseded and adds a new ADR. Prompts do not silently contradict an Accepted ADR.

## Consequences

- The catalog in `docs/patterns.md` and the prose in `docs/architecture.md` link here when they state a choice.
- A new pattern, a new external module that implements a cited algorithm, or a change to delete/index ordering needs an ADR in the same change as the code.
