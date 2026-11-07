# ADR-0035: Conic sketch measurements

Date: 2026-11-07. Status: Accepted for the Rust target.
Boundary: the kernel adapter's measured sketch curves and the desktop Measure panel.

## Context

The sketch benchmark requires the axis radii, axis diameters, length, and center
of a selected ellipse. The adapter already measures ellipse length through the
released Martensite analytic curve adaptor, but returns supplemental properties
only for circles. The desktop also repeats a circle lookup to name its angle.
Adding a second ellipse-only measurement engine would duplicate that boundary.

The Go reference keeps the Measure tool read-only and gives its panel a narrow
readout interface: `Oblikovati/app/measure_tool.go` and
`Oblikovati/head/ui/measure_dialog.go`. Ellipse definition and ownership follow
`Oblikovati/model/sketch/recipe_curve.go`. Current BDD scenarios determine the
properties that the Rust panel must report.

## Decision

Replace `SketchCircularMeasure` and `circular_measure` with one conic result and
one classification inside the existing kernel adapter measurement pipeline.
Use the public API's canonical `Curve2dType` values to distinguish full circles,
circular arcs, full ellipses, and elliptical arcs. These results contain analytic
center, both radii, and parameter span. They are internal, nonserialized adapter
values; they do not add or duplicate a public wire DTO.

Every curve still follows the same exact definition → native curve adaptor →
Martensite AbscissaPoint length query with convergence control. Use the SDK's
existing `precision::confusion()` metric resolution as the integration target;
this is also the tolerance used by OCCT DRAW's `length` command. Do not enlarge
or move model geometry, and do not return a new approximation of the curve.
The fixed-order overload gave 38.8046904979100 mm for axis radii 8 mm and 4 mm;
the convergence-controlled query gives 38.7537928821902 mm, agreeing with the
independent OCCT default-resolution oracle. OCCT at 1e-9 gives
38.7537928821907 mm. This is measured parity, not a claim that SDK iteration-limit
exhaustion certifies an achieved error for all possible input curves.

The desktop formats the returned properties
and performs no geometric classification or integration. Remove its repeated
circle lookup. Existing circle and arc rows retain their text and numeric values.
An ellipse reports both axis radii and diameters, its integrated length, and its
center transformed through the sketch plane. Tessellation is not an input.

## Deleted code and verification

Delete the circular-only metadata type, classifier, and host circle lookup.
There is no second length engine, fallback, or retained compatibility seam.
The classifier count stays one; no tolerance constant or fallback is added.

Verify the adapter through its public query and the panel through real measured
rows, including a rotated ellipse with axis radii 8 mm and 4 mm and perimeter
38.7537928821907 mm.
Use native keyboard and mouse scenarios to create and measure a recorded ellipse.
Retain circle, arc, line, distance, and angle checks to establish that existing
results remain unchanged. An inconsistent recorded benchmark value remains a
reported source defect; measurement must never manufacture inconsistent geometry.
