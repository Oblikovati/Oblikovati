<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Analytic edge-picking probe (#3967)

On the pinned Martensite revision `d051ff699982faba4ec2df3ee9a47e14f38dedbd`,
`edge_measure_probe.rs` passed five tests using `BRepExtremaDistShapeShape`.
The cases cover line segments, a circle, a cubic spline, and two circle
intersections. This is a limited diagnostic, not a general picking certificate.

The last case shows a design constraint: for a ray along positive Z and an edge
from `(0.05, 0, 1)` to `(0, 0, 10)`, the global nearest distance is zero at
Z=10. The first endpoint is already inside a 0.1-unit picking radius at Z=1.
A picker that treats the global nearest point as the frontmost eligible point
can produce the wrong depth and occlusion result. It needs the first point in
the pick volume, plus complete visibility and ambiguity handling. This probe
does not establish that Martensite lacks those operations.

To repeat, copy the probe into `crates/martensite/tests/` as
`edge_measure_probe.rs`, register it as a temporary test target in that crate's
manifest, then run `cargo test -p oblikovati-martensite --test edge_measure_probe`.
Remove the temporary registration and copy after the run.

The Go reference `app/raypicker.go` uses tessellated segments; its
`TestRayPickerSelectsEdge` and `TestRaySegmentDistance` passed at Go revision
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The checkout had untracked audit
Markdown files. Those tests do not certify analytic picking or stable names.
