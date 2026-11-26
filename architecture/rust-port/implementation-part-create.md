# Part Create implementation and verification

The shared `features/solid-model/CREATE-COVERAGE.md` index defines the
command scope. The scenario tree checked on 2026-11-26
contains 171 declarations across those 11 commands. An outline counts as one
declaration and can produce multiple executions. Extensions such as the
helical-spring sweep use the owning command and are additional scenario files.

| Command | Declarations |
| --- | ---: |
| Extrude | 56 |
| Revolve | 25 |
| Sweep | 21 |
| Loft | 22 |
| Coil | 7 |
| Emboss | 7 |
| Derive | 6 |
| Rib | 5 |
| Decal | 4 |
| Import | 12 |
| Unwrap | 6 |

## Extrude

The baseline run on `80cdf0e9c` executed all 74 current cases under Xvfb and
Lavapipe. Four passed (`extrusion-10`, `extrusion-11`, `extrusion-12a`,
`extrusion-12b`). Sixty-eight stopped on unbound steps, and two stopped on
assertions. The existing native named-parameter test also updates the
submitted mesh but fails its framebuffer comparison on the baseline and
after the binding changes.
The published 8 mm recipe has a literal distance. Its Tolerance menu item
currently requires an editable parameter, so `extrusion-12` stops before the
Tolerance window opens. Taper, surface and reference operations still need
their own end-to-end evidence.

The focused follow-up passed 10 executions and 71 steps on each of Wayland
and X11. It covers entry, direction changes and presentation, Cancel, zero
distance validation, reference selector validation, and the Parameters
table. See the [native run receipt](receipts/2026-11-26-extrude-entry-direction-3857/README.md).

## Sweep, Loft, and Revolve

The host exposes no runnable Sweep or Loft command at the baseline revision.
Qualify a straight-path Sweep and a two-section Loft through native preview,
commit, undo, redo and reopen before extending to guides, Boolean placement,
or spatial sections. Revolve has a persisted definition type and a geometry
probe, and still needs the desktop command and replay path. Each command
requires separate evidence for the regular, rejected and cancelled workflows.
