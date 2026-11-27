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
At that revision the published 8 mm recipe stopped before opening Tolerance
because its distance was literal.

The focused follow-up passed 10 executions and 71 steps on each of Wayland
and X11. It covers entry, direction changes and presentation, Cancel, zero
distance validation, reference selector validation, and the Parameters
table. See the [native run receipt](receipts/2026-11-26-extrude-entry-direction-3857/README.md).

The next pass covered `extrusion-06` on both desktops, including Undo, Redo
and two measured 8 mm edges. The named-parameter preview check now uses the
cyan preview palette and passes on both desktops. The wider native Extrude
suite passed all 14 tests under Xvfb.
See the [history and measurement receipt](receipts/2026-11-26-extrude-history-3857/README.md).

Distance A now opens the shared Tolerance inspector for the literal
8 mm fixture. The published and pinned cases 12–12c passed 36 native steps
per variant on Wayland and X11. They cover tolerance controls, cancellation,
the dimension prompt and the Parameters table.
See the [distance editor receipt](receipts/2026-11-27-extrude-tolerance-3870/README.md).

The [released-kernel taper probe](../../crates/martensite/tests/release_probe/extrusion_taper.rs)
uses the 40 mm × 30 mm rectangle and 8 mm depth with +5° and −5°. It
checks the analytic volume, the two cap widths, and a one-to-one native
history map across all faces, edges and vertices.

Edit Feature now accepts signed Taper A for single-sided normal extrusions.
The candidate drafts the side faces around the sketch plane. Native history
maps the existing face, edge and vertex roles to the drafted result, and
the adapter measures the new geometry before publishing it. The Part stores
Taper A in prism recipe version 2 and program version 40.
See the [native taper receipt](receipts/2026-11-27-extrude-taper-3870/README.md)
for scenarios 07–09, including history, package reopen and cancellation.

Extrude Join resolves the current body and sketch datums through the same
typed preparation for preview and acceptance. The shaded candidate shows
the existing solid with its added boss before OK. Acceptance retains one
solid and Undo restores the original solid and visible circle. See the
[native Join receipt](receipts/2026-11-27-extrude-join-3870/README.md)
for scenario 13 on Wayland and X11.

New Solid accepts a custom Body Name in the Extrude Output section. The
desktop identifies the new body by its exact key and renames it in the
same document transaction as the extrusion. Undo and Redo move the
feature and name together. See the
[native Body Name receipt](receipts/2026-11-27-extrude-body-name-3870/README.md)
for scenario 14 on Wayland and X11.

Selecting Cut from Default flips the extrusion inward. The preview shows
the opening on the named target before OK. Acceptance retains one solid,
Undo closes the opening, and Redo restores it. See the
[native Cut receipt](receipts/2026-11-27-extrude-cut-3870/README.md)
for scenario 15 on Wayland and X11.

Extrude Intersect uses the native Common operation and checks that the
result is one valid common solid with named topology. The saved
intersection key uses prism recipe version 3 and Part program version
41. See the
[native Intersect receipt](receipts/2026-11-27-extrude-intersect-3870/README.md)
for scenario 16, history, and package reopen.

Measure reads the selected cylinder's radius from its analytic surface
and reports the diameter and radius alongside its face area. The
[native Measure receipt](receipts/2026-11-27-extrude-intersect-measure-3870/README.md)
records the accepted Intersect's 7.069 mm² cap area, 3.000 mm diameter,
and 1.500 mm radius in scenario 17.

The desktop Through All termination enables Cut against one existing
solid and resolves the extent from the current target. The
[native Through All receipt](receipts/2026-11-27-extrude-through-all-3870/README.md)
checks both openings in the preview and accepted body.

Double-clicking a supported Extrude browser row opens its edit draft.
The panel shows the saved sketch-plane start and the body name, and
shows Match Shape checked and disabled on the saved closed profile.
The [native edit properties receipt](receipts/2026-11-27-extrude-edit-properties-3870/README.md)
covers the retained preview depth and cancellation.

The [measured direction outline](receipts/2026-11-27-extrude-measured-directions-3870/README.md)
checks signed preview bounds, parallel caps, and acceptance readiness
for Default, Flipped, Symmetric, and Asymmetric Extrude on both Linux
backends.

A numerical New Solid preview exposes a Distance A handle at its end
cap. Dragging it updates the distance expression through the same
validation and preview path as panel input. The
[native manipulator receipt](receipts/2026-11-27-extrude-manipulator-3870/README.md)
checks the dimension, field, and preview mesh after both kinds of input.

Extrude accepts length expressions that reference part parameters. The
[native driven-distance receipt](receipts/2026-11-27-extrude-driven-distance-3870/README.md)
checks that `Depth * 2` previews at 8 mm and rebuilds the same solid to
10 mm after `Depth` changes to 5 mm in the Parameters panel.

Measure in the Distance A menu can reference a visible sketch dimension.
The [native dimension-pick scenario](receipts/2026-11-27-extrude-dimension-pick-3870/README.md)
checks that clicking the 3 mm annotation inserts `d2` and keeps the
rectangular profile selected. A repeat pick is validated as `d2d2` and
Cancel leaves the source sketch intact.

The [shared profile scenario](receipts/2026-11-27-extrude-reused-profile-3870/README.md)
exercises selecting the visible sketch under Extrusion1 in the Model Browser.
After the second extrusion, the browser shows two solids and one shared
sketch. Both feature definitions reference that sketch.

For a single solid, Extrude's Operation selector offers icons and a
named dropdown. The [native selector scenario](receipts/2026-11-27-extrude-boolean-selector-3870/README.md)
checks Join, Cut, Intersect, and New Solid and verifies that changing the
presentation preserves the selected Join operation and its preview.

Advanced Settings offers Predict Boolean Operation. Clearing the option
keeps the current direction when choosing Cut or Intersect. The published
case checks the unchanged top face in Default direction and the opening
after manually selecting Flipped. [Native receipt](receipts/2026-11-27-extrude-boolean-prediction-3893/README.md).

Extrude starts with its preset row hidden. Advanced Settings reveals
the selector, add button, and gear. Named presets reuse numeric depth and
direction within the desktop session.
[Pinned visibility receipt](receipts/2026-11-27-extrude-preset-visibility-3870/README.md).

The Extrude breadcrumb offers Show Preview and Hide Preview. The live
input selections and distance remain in the panel while the viewport
switches between the committed solid and the edit preview. The shared
Preset Settings case also runs with the published rectangle.
[Shared Properties receipt](receipts/2026-11-27-extrude-shared-properties-3893/README.md).

Apply and Create New carries Extrude settings into a new draft and
clears its geometry picks. The published case applies an 8 mm Flipped
extrusion and cancels the following draft, leaving one solid.
[Native receipt](receipts/2026-11-27-extrude-apply-next-3893/README.md).

With Keep sketches visible on (+) checked, Apply and Create New commits
the extrusion and leaves its source sketch visible in a single undo step.
A viewport pick of the second closed region in the two-profile sketch
previews it without altering the committed extrusion.
[Sketch reuse receipt](receipts/2026-11-27-extrude-apply-reuse-3893/README.md).

Extrude Advanced Settings controls whether Enter commits the feature
or keeps the Properties panel open with the updated preview. Both
published examples exercise the active Distance A editor.
[Enter preference receipt](receipts/2026-11-27-extrude-enter-3893/README.md).

An accepted extrusion's Properties breadcrumb opens its source sketch
for editing. The active underline moves from the feature to the sketch,
and the extrusion preview clears when sketch editing begins.
[Source breadcrumb receipt](receipts/2026-11-27-extrude-source-breadcrumb-3893/README.md).

Tab and Shift+Tab move between the active asymmetric distance editors.
Typing in the focused second field updates the preview, and both
keyboard moves leave the draft open without adding a history entry.
[Value focus receipt](receipts/2026-11-27-extrude-value-focus-3893/README.md).

The published Extrude preset example verifies Save Current, reselecting
the saved distance, and renaming the preset while preserving its
stored settings. The outline's Revolve example remains open.
[Preset update receipt](receipts/2026-11-27-extrude-preset-update-3893/README.md).

Extrude refreshes the preview as soon as the final reference face is
selected. A [native From/To control](receipts/2026-11-27-extrude-between-faces-3870/README.md)
verifies caps on opposite faces of a block and an independent midplane
sketch. The published 3 mm radius fixture still needs Martensite's
translated-circle certification fix.

Extrude accepts origin planes and authored work planes from the Model
browser when a reference selector is active. The
[native plane control](receipts/2026-11-27-extrude-work-plane-3893/README.md)
checks the resulting cap and verifies that the saved recipe references
the authored plane by identity. On fractional-scale Wayland, the dock
layout keeps the Extrude panel clear of browser rows.

An Extrude can start on a parallel reference plane with a numeric
distance. The published fixture previews caps at 3 mm and 11 mm with
8 mm of depth. Commit and reopen retain the start plane by identity.
[Native receipt](receipts/2026-11-27-extrude-distance-from-plane-3893/README.md).

Face extension is pending. The existing planar reference path resolves
the supporting plane for From and To. Disabling extension needs a
certified comparison of the projected profile against the trimmed
reference face, including holes and curved trim edges. Point picks and
tessellated face bounds alone cannot establish that coverage.
[Plane resolution](../../crates/model/src/feature/sketch_prism/extent.rs).

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

A Closed Loop edit on the straight center line checks the native guide wire,
reports the periodicity requirement, and restores the previously saved Loft
when canceled. The [closed edit receipt](receipts/2026-11-27-loft-closed-edit-3917/README.md)
records the dialog and viewport checks on Wayland and X11.

### Circular surface section conditions

The first section of a coaxial circular surface Loft can change from Free
to Angle in the Conditions tab. The default 90° angle and unit weight
change the takeoff tangent. Editing either value rebuilds the private
preview. The native adapter revolves a cubic meridian and binds the two
generated trims to the authored section IDs. Program recipe version 39
saves the condition with the existing Loft identity.

The [native receipt](receipts/2026-11-27-loft-conditions-3917/README.md)
covers 09 and 11 on Wayland and X11, including package reopen and the
published and pinned scenario variants.
