# ADR-0029: Released sketch geometry ownership

Date: 2026-11-05. Status: Accepted for the Rust target design.
Implementation acceptance is tracked by
[#3814](https://github.com/Oblikovati/Oblikovati/issues/3814).
Boundary: owned sketch geometry to native analytic construction and queries.
Supersedes ADR-0024; its historical record remains unchanged.

## Decision

Use Martensite v0.8.0 at
`59aca38960bdfbaa75a07a80d9b2de522005ddac` for authored planar curves,
conic construction, spline construction, intersections, sampling, and
camera-line intersection with a sketch plane. Use the versions of each
individual toolkit in that release, rather than assigning the release number
to every Cargo package.

Delete the RustyCAD bindings in these operations and the direct
`rustycad-tkg2d` and `rustycad-tkgeomalgo` dependencies. Each operation keeps
one native construction and query path. A native failure remains a typed
failure at its source entity; it cannot select the previous SDK as a fallback.

Keep application-owned inputs and reports. Construct native handles inside the
adapter and return coordinates, parameters, source identity, construction
limits, and measured residuals. Mutable toolkit handles do not cross the
application interface. Public wire types and client calls remain owned by
Oblikovati.API.

Control and fit splines, closed curves, and named offsets use the released
native sketch constructor. Map the existing API fit policies to its native
policy. Copy accepted poles, weights, knots, and multiplicities into the owned
spline definition. Keep fit stations, source identity, achieved fit and seam
residuals, and separate fit, seam, derivative, and endpoint limits. Derivative
handle authoring remains separate model and API work.

Keep the signed offset convention from ADR-0016 and the point ownership from
ADR-0017. Sampling, display, and analytic queries use the same constructed
recipe. Sampling deflection remains independent of modelling tolerances;
analytic queries never consume display samples. Invalid entities retain their
local display declines or typed query errors.

## Solid geometry boundary

The production solid adapter retains its qualified SDK until Martensite's
analytic sketch-region and solid-result gates pass. The remaining gates are
tracked by Martensite #386 and #390. This is a separate operation boundary,
not a fallback for a failed v0.8 sketch query. This decision does not qualify
Extrude, solid properties, or topology provenance.

## Verification and deletion

Retain the Go fit recipes, the six construction policies, and the 169 OCCT
construction samples used by ADR-0024. Preserve their numerical assertions.
Run the existing spline, conic, intersection, offset, incidence, projection,
preview, and display corpus against the new bindings. The full local gate and
live native rectangle, parameter, history, save, and reopen acceptance must
pass before the first push.

A v0.8 projection probe confirms that orthogonal roots alone do not define a
bounded closest point: a finite segment can have no orthogonal root, and a
trim endpoint can be nearer than every returned stationary point. Future
picking must use the general bounded minimum, including boundaries, through
this same released construction path.

This migration changes no host formulas, recognizers, tolerance constants,
fallback sites, or geometry type assertions. It removes the previous 2D
bindings and their two direct dependencies. Runtime qualification depends on the checks above; the implementation ledger
records their results.
