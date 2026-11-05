# Sketch frame orientation — partial R3

Tracking: [#3736](https://github.com/Oblikovati/Oblikovati/issues/3736).
`SketchPlane::with_orientation` ports the frame calculation used by Go
`addin/router/sketch.go` (`orientSketchPlane` and `orientedAxes`) at migration
baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The caller supplies a current resolved reference direction. One operation projects
it into the plane, selects X or Y, applies reversal, and completes a right-handed
frame. It retains the original origin unless an explicit finite origin is supplied.
The frame's normal and metric validation policy are preserved. The existing frame
constructor checks the result before it can be returned.

Direction normalization scales by the largest finite component first. This avoids
overflow and underflow for large and subnormal direction vectors. A double cross
computes the planar projection without subtracting nearly equal normal components.
The sine of the angle to the normal must exceed the frame's `angular_deviation`.
Zero, non-finite, and ill-conditioned directions produce a named error. No error
returns a stale base frame. This conditioning check uses the existing frame metric
policy; it is not a kernel topology predicate or a modelling tolerance.

Tests cover both axis choices and signs on all origin planes, the Go 30-degree
meridian case, finite direction scales from the smallest subnormal to `f64::MAX`,
near-perpendicular conditioning, invalid directions/origins, saved-frame restore,
and coordinate round trips. An independent calculation in the original local
basis checks 256 generated oblique orientation cases. The four Go router orientation
tests pass on Linux at the fixed baseline.

The numerical frame operation is now used by the owned orientation state below.
The owned line-axis catalogue is implemented in [mixed datums](implementation-work-geometry.md).
The edge catalogue and shared `sketch.create` routing are still required. Native and kernel acceptance remain open.

## Owned orientation recipes and current resolution

Tracking: [#3737](https://github.com/Oblikovati/Oblikovati/issues/3737).
Placed sketches retain the selected origin axis or a stable feature-axis ID,
X/Y selection, reversal, and optional origin. The model definition is internal;
it does not redeclare the public API DTO or method names. Origin axes resolve
directly. Feature axes now use the current owned mixed datum evaluation. An
injected `SketchAxisResolver` remains available for explicit callers. Neither
route returns last-good geometry. The edge catalogue remains open.

Every query resolves the current base plane and axis, then calls the same checked
frame operation. Neither the initial direction nor a derived frame is stored.
A moved base or invalid axis produces the current result or a named placement
error. Local geometry and its solver verification remain independent of this
model-space placement. Axis feature IDs join the sketch's exact read footprint.

Sketch recipe version 3 supports orientation. Versions 1 and 2 keep their previous
layout when orientation is absent and refuse orientation content when present.
Zero feature IDs and non-finite origins are refused before state adoption.
Exact authored comparison includes origin bits, so signed zero is an edit.
Tests cover changed directions, current base-plane degeneracy, unavailable axes,
all origin axes, invalid/versioned recipes, read attribution, option changes,
undo/redo, saved baselines, reopen, and independent file copies.
