# M19 · F07 — UI: Materials window (browser, editors, assign, readout)

> **Backfilled 2026-06-04 from shipped code.** See REPORT.md D-03.

## Scope (in)

The head Materials/Appearance UI: asset browser, appearance editor (albedo/metallic/
roughness/…), material editor, assign-to-selection, physical-properties readout, and the
Preferences integration.

## Code (as built)

`head/ui/{materials_window.go,appearance_editor.go,appearance_tab.go,preferences_window.go}`
(+ `materials_window_test.go`), `app/materials.go` session bridges.

## DoD note

Per CONVENTIONS "Definition of Done", verify there is an **end-to-end test** driving
assign → live viewport recolor → readout (the milestone's exit criterion). `head` tests
are **not run in the `CGO_ENABLED=0` CI suite** — see REPORT.md §8 (recommend a head-test
CI gate before grading U✅).

## PBIs

| PBI | Title | Grade |
|-----|-------|-------|
| [PBI-198](PBI-198-materials-ui.md) | Materials window + editors + assign + readout | M✅ U🟦 (e2e unverified, see §8) |
