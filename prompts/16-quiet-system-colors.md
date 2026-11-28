# Prompt 16 — Quiet system colors

Read `docs/architecture.md` (Review window), `docs/use-cases.md` (UC-2), `docs/adr/0013-native-review-window.md`, and `prompts/15-open-copy.md`.

Prompt 15 is already in effect: each copy column has Open, and Open does not record a decision.

## Outcome

The window sits beside Finder and a browser without looking louder than they do. The accent color fills only the committing actions: Apply, and “Apply to all files of this type in …”. Every other control uses the control color, the text color for its label, and a separator-colored border. That includes Choose folder, Start scan, Dry run, Skip set, Open, Keep, Delete, and Skip.

Borders and the rule between the list and the columns use the separator color. They do not use the text color. Titles use the text color. Paths, sizes, hints, and row status use the secondary text color. Columns and the preview well use the surface color, which is close to the window background.

A role is the system color when the operating system provides it, including its alpha. Otherwise it comes from the light or dark fallback below. The fallbacks are the Aqua and Dark Aqua resolutions of Apple’s semantic colors measured on this Mac. Apple documents these names, not these numbers, and says the numbers can change in a later release. A missing accent does not change the background, the surface, or the text. On-accent is not read from the operating system; it stays white. The window still repaints when the appearance changes. Gio’s indigo `#3F51B5` is not a fallback and is not used unless the system accent is that exact color.

Quiet buttons use the control color, not the accent. The current window drops alpha and paints every button with `controlAccentColor`, which is why the chrome looks harsher than Finder.

Light fallback, Aqua:

| Role | Apple color | Resolved |
| --- | --- | --- |
| Background | `windowBackgroundColor` | `#FFFFFF` |
| Surface | `underPageBackgroundColor` | `#F6F6F6` |
| Control | `controlColor` | `#FFFFFF` |
| Text | `labelColor` | `#000000` at 85% |
| Secondary | `secondaryLabelColor` | `#000000` at 50% |
| Separator | `separatorColor` | `#000000` at 10% |
| Accent | `controlAccentColor` | `#007AFF` |
| On accent | none | `#FFFFFF` |

Dark fallback, Dark Aqua:

| Role | Apple color | Resolved |
| --- | --- | --- |
| Background | `windowBackgroundColor` | `#1E1E1E` |
| Surface | `underPageBackgroundColor` | `#282828` |
| Control | `controlColor` | `#FFFFFF` at 25% |
| Text | `labelColor` | `#FFFFFF` at 85% |
| Secondary | `secondaryLabelColor` | `#FFFFFF` at 55% |
| Separator | `separatorColor` | `#FFFFFF` at 12% |
| Accent | `controlAccentColor` | `#007AFF` |
| On accent | none | `#FFFFFF` |

macOS reads those semantic colors under the current appearance and keeps the alpha. Windows still reads light or dark from `AppsUseLightTheme` and the accent from `DwmGetColorizationColor`; the other roles stay on the fallback. Linux still reads `color-scheme` and the optional accent and does not link GTK. Do not invent a brand green.

## Acceptance tests

- `TestAcceptance_QuietFallbackIsNotBlackOnWhite`: the dark fallback background is `#1E1E1E`, its surface is `#282828`, its separator is white at 12% and is not the text color, and its control color is white at 25%. The light fallback matches the Aqua table above. Text and secondary text keep their alpha.
- `TestAcceptance_MissingAccentKeepsTheQuietSurface`: a dark appearance with only a system background set keeps the dark fallback surface, text, secondary, separator, and accent.
- `TestAcceptance_SystemPaletteFollowsAppearance`: extend the existing cases so each new role follows the same rule as background and text. A supplied separator replaces the fallback separator and leaves the accent alone.

Tests pass the appearance and the system colors in. They do not read the developer’s display and do not open a window.

## Notes

Do not change which files are listed, the keeper rank, Open’s behavior, or the release archive names. Update the Review window prose and its diagram in the same change.
