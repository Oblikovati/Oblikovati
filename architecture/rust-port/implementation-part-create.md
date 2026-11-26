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
assertions.
The published 8 mm recipe has a literal distance. Its Tolerance menu item
currently requires an editable parameter, so `extrusion-12` stops before the
Tolerance window opens. Taper, surface and reference operations still need
their own end-to-end evidence.

The focused follow-up passed 10 executions and 71 steps on each of Wayland
and X11. It covers entry, direction changes and presentation, Cancel, zero
distance validation, reference selector validation, and the Parameters
table. See the [native run receipt](receipts/2026-11-26-extrude-entry-direction-3857/README.md).

The next pass covered `extrusion-06` on both desktops, including Undo, Redo
and two measured 8 mm edges. The named-parameter preview check now uses the
cyan preview palette and passes on both desktops. The wider native Extrude
suite passed all 14 tests under Xvfb.
See the [history and measurement receipt](receipts/2026-11-26-extrude-history-3857/README.md).

## Sweep, Loft, and Revolve

The [release probes](../../crates/martensite/tests/release_probe/create.rs)
check the first Sweep and Loft fixtures' geometry on Martensite v0.14.2.
The straight Sweep produces a cylinder of volume `0.75π cm³`. Ruled and
smoothed Loft both produce a frustum of volume `1.75π cm³`. Each result is
a valid solid with two cap faces and a side face attributed to the source
curves. Both Loft section edges generate the same side face.

Sweep and Loft still require persistent input identities, selection,
preview, commit, undo, redo and reopen in the host. The probes cover the
straight circular Sweep and two coaxial Loft sections. Guided, Boolean and
spatial inputs still need qualification. Revolve has a persisted definition
type and a geometry probe, and still needs desktop authoring and replay.
