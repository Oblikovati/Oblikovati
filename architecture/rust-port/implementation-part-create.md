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
covers `sweep-01` and `sweep-02` on Wayland and X11. Profile and path repicking, twist, guide references,
Boolean and surface operations remain open.
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
driver. Existing untapered documents still use version 28.

Sweep Properties uses the same taper parser for creation and feature edits.
Edit Feature reopens the saved profile, path, and taper. Each input revision
gets a new preview, and OK requires certification of that revision.

The [desktop taper receipt](receipts/2026-11-26-sweep-taper-edit-3917/README.md)
covers all eight steps of `sweep-03` on Wayland and X11, including the
change from the tapered preview to straight sides and the saved zero angle
on the same Sweep.

### Sweep orientation

Properties offers Icons and Dropdown selectors with Follow Path, Fixed,
and Guide. A Guide draft without a reference clears the preview, requests a
rail or surfaces, and disables OK. Fixed translates the profile along a
straight path while keeping its frame, including for an oblique path, and
hides the taper field. Fixed orientation is authored in version 30 of the
Part program. Its saved sources and body reference keys survive reopen.

The [orientation receipt](receipts/2026-11-26-sweep-orientation-3917/README.md)
covers the Guide validation and Fixed edit on both Linux backends. Guide
reference picking and guided geometry remain open. The single-selection
checkbox is editable for the current draft and resets when the Sweep is
reopened. See the [checkbox receipt](receipts/2026-11-26-sweep-draft-option-3917/README.md).

### Clear a saved Sweep path

Clear Path keeps the selected profile in the draft, requests a curve or
edge, removes the Sweep body from the preview and disables OK. Restore
Path puts the saved reference back in the draft. Cancel shows the original
body again, and reopening the feature restores its references and taper.
See the [native receipt](receipts/2026-11-26-sweep-path-cancel-3917/README.md).

### Sweep twist

Follow Path accepts a signed twist angle or an angular parameter. The
released kernel can evaluate it for a single circular profile when the
straight path starts at its center and is normal to its plane. Rotation
preserves the circular sections, including those of a tapered Sweep. The
adapter requires that geometry before using the same analytic result and
face ownership. Nonzero twist outside that geometry is declined. The
Part program stores the twist driver in version 31.

The [native receipt](receipts/2026-11-26-sweep-twist-3917/README.md)
covers accepting and reopening a 90-degree twist and correcting an
invalid expression. The [replay test](../../crates/martensite/tests/release_probe/sweep_sources/replay.rs)
checks the authored value, body references, and analytic volume after
package reopen.

### Reselect a Sweep profile

Clear Profile keeps the saved path in the draft, hides the Sweep body, and
temporarily shows the original profile sketch for picking. A viewport click
inside a certified bounded region on that sketch supplies the directed
boundary. The preview and OK return after the new draft is evaluated. Empty
space, edge-on views, and ambiguous hits leave the selection open.
See the [native receipt](receipts/2026-11-26-sweep-profile-pick-3917/README.md).

### Sweep presets

The preset control saves certified orientation and angle settings to the
desktop preference profile. Bare numeric angles are saved with their
current angular unit. Choosing a preset reapplies those settings to the
current draft without changing its profile or path. The preset file is
independent of the Part and its Undo history.
See the [native receipt](receipts/2026-11-26-sweep-presets-3917/README.md).

### Recent Sweep twist expressions

The Twist field offers recently accepted changes from the current desktop
session. Choosing an expression updates the draft and validates it against
the current geometry. Cancel discards that draft. The [native receipt](receipts/2026-11-26-sweep-recent-twist-3917/README.md)
covers accepting 45 degrees, reusing the original expression, and
cancelling without editing the saved feature or body.

### Sweep New Solid and Join

New Solid shows the next body name and creates an independent body.
Join uses the sole eligible target and keeps its body reference when it
replaces the target with the combined solid. The version 32 Part program
stores the exact target key alongside the Sweep sources. The
[replay probe](../../crates/martensite/tests/release_probe/sweep_sources/replay.rs)
checks the target dependency, retained identity, and analytic volume.
The [native receipt](receipts/2026-11-26-sweep-join-3917/README.md)
covers creating Solid2 and editing the same Sweep to Join.

### Sweep Cut

Cut subtracts the swept volume from the sole eligible body and retains
its reference. The version 33 Part program stores the operation with the
target key for replay. When an edit would remove no material and the face
count stays the same, OK opens the Edit Sweep Feature warning with a
collapsed Details section. Cancel closes the edit and leaves the saved
Join, body, and document generation intact.
The [native receipt](receipts/2026-11-26-sweep-cut-3917/README.md) covers
the unchanged edit on Wayland and X11. The published inward cylinder
fixture is blocked at profile certification by
[Martensite #619](https://github.com/Oblikovati/Martensite/issues/619).

### Surface Sweep

Surface output sweeps the certified contour as a wire. The result is an
open shell with a named side face and two named circular trims. The part
program stores the output type from version 34. Reopening a part retains
the surface references and its independent solid.

The [release probe](../../crates/martensite/tests/release_probe/sweep_sources/surface_replay.rs)
checks replay, tessellation, and stable references. The builder probe
matches the 5 mm radius and 30 mm path in the published outward fixture.
The desktop can acquire the visible profile and path automatically or
pick them in the viewport. It lists the result separately under Surface
Bodies. The [native receipt](receipts/2026-11-27-sweep-surface-3917/README.md)
covers the published explicit selections on Wayland and X11.
The same receipt covers the incomplete Guide draft. A browser sketch
click leaves Path and Guide awaiting geometry and keeps OK disabled.

### Loft entry and section selection

The Part ribbon opens a dockable Loft draft. An empty Part shows geometry
guidance with OK and Help. With visible circular sketches, viewport picks
retain the authored point or profile curve identity and list the sections
in pick order. The [native receipt](receipts/2026-11-27-loft-section-selection-3917/README.md)
covers the first two published scenarios.

### Loft solid creation and editing

Two circular sections drive the released section builder. The Part recipe
stores their identities in order. The [solid receipt](receipts/2026-11-27-loft-solid-3917/README.md)
covers the native preview and body identity through a package round trip.
The [mapping receipt](receipts/2026-11-27-loft-mapping-3917/README.md)
covers switching between automatic and manual transition mapping in a draft.

Double-clicking Loft1 in the browser restores both source rows and evaluates
the existing solid for its edit preview. Cancel discards Closed Loop and
Merge Tangent Faces draft changes and restores the committed presentation.
The [edit receipt](receipts/2026-11-27-loft-reopen-3917/README.md)
covers cancellation and reopening on both Linux display backends.

The [path mode receipt](receipts/2026-11-27-loft-path-modes-3917/README.md)
covers choosing both Center Line modes in an existing solid edit and restoring
the Rails list and preview.

### Loft surface

Surface output keeps the side wall and both named circular trims. The saved
Part recipe records the output kind at program version 36. The
[surface receipt](receipts/2026-11-27-loft-surface-3917/README.md) covers the
preview, Surface Bodies entry, and body identity through save and reopen.

The [point set receipt](receipts/2026-11-27-loft-point-set-3917/README.md)
covers adding an incomplete manual set in a surface edit and cancelling while
retaining the committed surface.

### Loft Center Line

The solid Loft accepts a straight sketch line that connects both circular
section centers. The authored line and its sketch participate in replay and
stable body naming. Part recipe version 37 stores the guide identity. The
[center-line receipt](receipts/2026-11-27-loft-center-line-3917/README.md)
covers native picking, preview, commit, and package reopening.

### Loft intermediate section

A private edit projects the viewport pick along the saved Center Line.
The released through-sections builder constructs a checked three-section
solid from the saved profile circles and the intermediate circle. The
intermediate position is normalized along the guide and its size multiplies
the area of the interpolated section. Part recipe version 38 preserves both
values. The [native receipt](receipts/2026-11-27-loft-placed-3917/README.md)
covers the viewport pick, annotations, two acceptance steps, and package replay.

The [section edit receipt](receipts/2026-11-27-loft-section-edit-3917/README.md)
covers reopening at the projected viewport point, doubling the area and
checking the larger preview, then moving the point along its 20 mm guide.
