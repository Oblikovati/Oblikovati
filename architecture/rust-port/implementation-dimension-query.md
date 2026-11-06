# Dimensional sketch inspection

Tracking: [#3782](https://github.com/Oblikovati/Oblikovati/issues/3782).

The host consumes the released API's dimension DTOs, method constant, and typed
`client.sketch().dimensions(&args)` call. Both file hosts use one immutable query.
The model now supplies an authored dimensional enumeration with backing parameter
IDs. Geometric and dimensional indices remain separate; structural arc and
projection sources are outside both authored collections.

All current dimensional forms are reported: three distance orientations, radius
and diameter on circles or arcs, arc length, offset and linear-diameter offset,
line angle, three-point angle, and near/far tangent distance with its diameter
option. Kinds use API constants, including the canonical `offsetDim` spelling.
Names and expressions come from current parameter state. Missing, failed, or
incompatible parameter sources produce named declines.

The `value` field is a geometric measurement, as in Go's
`DimensionConstraint.Measured`. It is independent of the parameter target. A
nonconverged sketch can report a measured value that differs from its expression;
the query neither solves nor claims that the target was achieved. Measurements
read the stored geometry and reuse the solver's distance, angle, arc-length,
circular projection, and tangent-offset arithmetic. Lengths are converted from
working units to centimetres; angles remain radians. Undefined orientations and
unrepresentable results are errors, with no zero substitute.

The existing Rust arc model retains the full authored sweep for major arcs,
including its previously recorded difference from Go's minor-arc measurement
(#3713). The query uses that same model measurement. Driving dimensions report
`driven: false`; the later [dimension edit port](implementation-dimension-edit.md)
adds reference mode and reports its measured geometry with `driven: true`.
Annotation placement is stored and returned; see
[dimension placement](implementation-dimension-placement.md).

Tests cover every form and metadata field, independent analytic values, measured
values that differ from targets, rotated/scaled and reversed geometry, zero point
distance, major arcs, degenerate orientation, conversion overflow, mixed
collections, empty arrays, both hosts, query immutability, current source changes,
failed sources, wrong owners and payloads, history, save/reopen, and copy. The
pinned Go model and router tests supply independent reference evidence.

The full local gate covers the Rust workspace present in this tree. Native
application rendering, dimension creation, and feature/kernel acceptance remain
open. The [kernel package intake](implementation-kernel-release.md) records
released toolkit consumption separately.
