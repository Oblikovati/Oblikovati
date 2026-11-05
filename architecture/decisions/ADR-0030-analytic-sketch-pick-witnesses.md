# ADR-0030: Analytic sketch pick witnesses

Date: 2026-11-05. Status: Accepted for the Rust target design.
Boundary: analytic point-to-curve candidates for native sketch picking.
Tracking: [#3816](https://github.com/Oblikovati/Oblikovati/issues/3816).

## Decision

Reuse the released curve construction path from ADR-0029. Query native extrema
on each C2 continuity interval and include the finite interval boundaries.
Stationary projections alone are insufficient: a segment may have none inside
its range, and a trimmed arc endpoint may be nearer than every stationary point.
Internal spline knots are boundaries of the same general domain partition.

Do not add a curve-type picker or another construction engine. Native analytic
and general extrema remain inside the kernel's representation dispatch. A
failed native interval declines the query with its input context; it cannot
silently return only the boundaries.

Re-evaluate native stationary candidates on the curve. Check the position
residual, the normal stationarity residual in length units, and the native
minimum classification. Return owned coordinates, native parameters, distance,
and witness residuals. No mutable kernel handle escapes. Choose candidates by
a total order of distance, parameter, coordinates, origin, and residuals. Native completion is
not advertised as a certificate of root completeness.

Use one versioned options value with explicit native and witness limits and a
bounded interval budget. Defaults use the release's existing precision policy;
callers can supply achieved sketch precision. Picking does not change geometry
or feed display deflection into modelling tolerances.

## Desktop integration and deletion

The editor will resolve point and curve witnesses to existing model identities,
then mint checked session picks. The same session state must drive rendering
and the canonical selection query. Display samples never resolve references.
Placement tools retain their inputs. Constraint tools consume the accepted
selection through the normal host operations.

Delete the editor's ignored ordinary-left-click path when the owner route is
connected. There is no previous closest-curve engine to retain as a fallback.
ADR-0009 remains the authority for UI ownership and selection policy.

## Acceptance

Use analytic endpoint and conic oracles, the existing rational spline and Go
curve fixtures, and near-degenerate inputs. Cover trim endpoints, internal
spline knots, offsets, periodicity, conditioning, invalid inputs, and repeated
results. Native mouse/keyboard tests must observe the same entity selection in
the API and viewport before constraint tools are accepted. The full local gate
is required before the first push.

The operation adds no geometry recognizers, tolerance constants, fallback
sites, or geometry type assertions. The interval budget is a resource limit.
Implementation and remaining live acceptance are tracked under #3816.
