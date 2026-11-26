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

### Sweep source resolution

The versioned Sweep recipe records the profile boundary, both source sketch
IDs, and the ordered, directed path curve IDs. Resolution reads both solved
sketches and attributes their datum and parameter dependencies. The adapter
resolves the selected closed region, lifts the analytic path from its
current sketch plane, and checks wire traversal before calling Martensite.
The result must have a valid solid, measured B-rep tolerances within the
configured cap, and exactly one construction role per face.

The [source tests](../../crates/martensite/tests/release_probe/sweep_sources.rs)
round-trip the recipe, rebuild the cylinder after a path length edit, and
reject a replacement path even when it occupies the same coordinates. A
disconnected path reports the offending curve and returns no solid.

### Sweep history and persistence

Ordered Part replay produces a named solid for a supported Sweep. The
document stores both sketch selections in Part program version 28. The
body and faces use authored source roles. Edges and vertices use their named
incident entities. Replay declines ambiguous topology instead of assigning
keys from kernel enumeration order.

The [replay test](../../crates/martensite/tests/release_probe/sweep_sources/replay.rs)
checks the same body, face, edge and vertex keys through full replay, suffix
replay, Undo, Redo, a path-length change and package reopen. It also checks
the volume and measures the two circular edges and the straight seam.

### Straight-path desktop Sweep

Part → Create → Sweep discovers a unique visible closed region and an
independent single-line path. The preview uses the current Part generation
and the new-solid palette. OK commits the same inputs through `features.add`,
and Undo restores the source sketches' visibility. An empty Part opens the
geometry tip and Help.

The five Part Create icons already used in the desktop now match the feature
SVGs for Extrude, Revolve, Sweep, Derive, and Import. The drawing code applies
the named paint classes in Dark, Light, and High Contrast.

The [desktop receipt](receipts/2026-11-26-sweep-entry-history-3917/README.md)
covers `sweep-01` and `sweep-02` on Wayland and X11. Sweep input selection and
editing, twist, guides, Boolean and surface operations remain open.
Loft still requires authored section identities and host integration.
Revolve has a persisted definition type and a geometry probe, and still
needs desktop authoring and replay.

### Taper on a straight Sweep

The typed Sweep API accepts a taper angle or named angle parameter. A
nonzero taper builds an analytic frustum from a circular profile and a
single path normal to its plane. Positive angles widen the end section and
negative angles narrow it. Zero continues to use the original pipe builder.
Collapsed sections fail validation.

The [geometry tests](../../crates/martensite/tests/release_probe/sweep_sources/taper.rs)
check signed angles against analytic volumes and verify that a parameter
edit invalidates the Sweep. The
[history tests](../../crates/martensite/tests/release_probe/sweep_sources/replay.rs)
verify stable body, face, edge, and vertex keys through package reopen and
after changing the taper to zero. Part program version 29 stores the taper
driver. Existing untapered documents still use version 28. The Sweep
Properties editor still needs the taper field and Edit command.
