# Checked work-axis line values — partial R3

Tracking: [#3738](https://github.com/Oblikovati/Oblikovati/issues/3738).
`feature::work_axis::WorkAxisLine` ports the checked source value used by Go
`model/feature/work_axis_point.go` and `serialize_work_axis.go` at baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The immutable value requires a finite working-unit origin and a nonzero finite
direction. The direction is dimensionless. Construction shares the sketch-frame
normalization function, including finite extreme and subnormal scales. Extraction
of that function preserves its existing arithmetic and comments; it does not add
a second normalization algorithm. The checked direction can feed the same sketch
orientation operation.

The version-1 line recipe stores the origin and unit direction. Restore uses the
existing frame unit-deviation policy and keeps validated components unchanged.
It does not normalize them again. Exact source comparison includes signed origin
and direction bits. Invalid versions, non-finite origins, zero/non-unit saved
directions, and unknown serialized fields are refused with named diagnostics.

Tests cover Go line-axis values, scale extremes and reversed signs, invalid input
causes, repeated serialization/restore without bit drift, signed origins, unit
policy limits, and consumption by sketch orientation. The Go line-axis and line
serialization tests pass on Linux at the fixed migration baseline.

This value has no feature ID allocator, collection, replay engine, or I/O. Line
axes now join the existing ordered datum program and permanent feature namespace;
see [mixed datum evidence](implementation-work-geometry.md). Two-point and
plane-intersection definitions and public axis/sketch creation commands remain
open. Native and kernel acceptance are not established by these tests.
