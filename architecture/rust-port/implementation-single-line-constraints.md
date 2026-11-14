# Single-line alignment and geometric indices

[Issue #3778](https://github.com/Oblikovati/Oblikovati/issues/3778) adds the
constraint representation and enumeration prerequisite for inferred line creation
under [issue #3775](https://github.com/Oblikovati/Oblikovati/issues/3775).

`PlanarConstraint::LineHorizontal` and `LineVertical` retain an owned line as
their authored operand. The existing `Horizontal` and `Vertical` remain two-point
alignments. This distinction matches Go issue #1871 and the canonical API's
`horizontal`/`vertical` versus `horizontalAlign`/`verticalAlign` kinds.
Ownership validation refuses points, circular curves, and missing lines.

Compilation resolves the line's current endpoints inside the shared preparation
path. Both line and point forms use the same coordinate equation, variable order,
Jacobian, diagnostics, and solver. No parallel numerical implementation is added.
Their native recipe tags retain the different authored operands. Older recipe
content remains readable; unknown or invalid line operands fail restoration.

`Sketch2D::geometric_constraints` filters the existing ordered constraint list
through the shared dimension-parameter classification. `geometric_constraint_index`
returns the zero-based position in that projection and refuses missing or
dimensional identities. Constraint IDs stay independent of enumeration positions.
Deleting an earlier geometric relation changes subsequent indices, not their IDs.

The existing Rust point-to-line `Offset` is a dimensional relation, matching Go
`AddOffsetDim`; it is not the unported two-line Go geometric offset constraint.
The index corpus covers distance, radius, diameter, arc length, offset, tangent
distance, two-line angle, and three-point angle, interleaved with geometric
relations. Arc invariants are numerical sources and do not enter this authored
geometric enumeration.

Model tests check distinct saved operands, invalid ownership, bit-identical
solutions between line and point forms, rank and freedom, fixed-endpoint conflicts,
parameter-driven DOF-zero geometry, degenerate coordinate semantics, and twelve
generated orientation/scale cases. Persistence tests check both line kinds and
malformed native operands. Named file owners verify current DOF-zero geometry,
undo/redo, save/reopen, and copy. Their fixture places an offset dimension before
a new line relation, so full-list position 4 must project to geometric index 3.
Metric checks use the solver policy for the fixed unit-length fixture.

The Go single-line horizontal/vertical corpus passes at the
[source baseline](source-map.md). Validation is local Linux. This work does not
add `sketch.addConstraint`, constraint query rows, inferred line creation, a native
viewport, or kernel acceptance. Atomic creation and canonical applied-inference
responses remain #3775; the full application port remains open.
