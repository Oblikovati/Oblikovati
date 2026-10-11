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

This is the numerical frame operation. Live axis/edge reference resolution,
persisted associative orientation, and shared `sketch.create` routing are still
required. The future resolver must call this same operation with current inputs
on each recompute and report invalid references through feature health. It must
not freeze the first resolved direction or fall back to the base plane. Native
and kernel acceptance remain open.
