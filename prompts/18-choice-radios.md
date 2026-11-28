# Prompt 18 — Choice radios

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-2, UC-3), and `prompts/15-open-copy.md`.

Prompt 15 is already in effect: Open launches the copy and records no decision. Prompt 16 is already in effect: Open is a quiet button. Prompt 17 is already in effect: the progress label is relative to the scan root.

## Outcome

On each copy column, Open stays a button. Keep, Delete, and Skip are one radio group, not three buttons. The selected radio is the choice the owner can see.

The selected value is the recorded decision when one exists. Otherwise it is the suggestion: Keep on the suggested keeper, Delete on each path in that group’s delete list. A group with no recommendation, including a tie and every similar group, has no radio selected. A similar group still has no delete suggestion. Selecting a radio records that choice for the absolute path and replaces an earlier choice for the same path. Skip set and the folder action still record the same decisions they record today, and the radios show those decisions.

## Acceptance tests

- `TestAcceptance_SuggestedChoiceIsTheDefault`: an exact group whose keeper is the plain name shows keep for that path and delete for the generated copy. The session still has no decisions.
- `TestAcceptance_RecordedChoiceBeatsTheSuggestion`: a recorded keep on the suggested delete shows keep.
- `TestAcceptance_NoSuggestionLeavesTheRadiosClear`: a tied exact group and a similar group show no choice for every member.

Tests use the review model and the session. They do not open a window.

## Notes

Do not change DefaultKeeper, the similar-group delete rule, Open, which files are listed, or the release archive names. Update the Review window prose and its sequence diagram in the same change.
