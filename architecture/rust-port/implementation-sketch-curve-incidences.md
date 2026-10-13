# Owned sketch curve incidences — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/edit_trim.go`, `edit_trim_curves.go`, and
`edit_trim_freeform.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
This is the analytic query boundary needed by line trim and extension.
It does not yet change entities or expose those commands through automation.

`planar_curve::sketch::intersect_sketch_line` borrows an owned sketch and returns
independent curve definitions and complete native result records for each target.
`SketchLineDomain::Segment` trims every line operand to its authored endpoints.
`Support` uses infinite supports for both the selected line and other lines,
as the Go extension operation does. Circle and ellipse operands remain complete
conics; arcs retain their authored direction and actual bounded sweep.

Curves are resolved from shared point identities. Ellipses retain their major
axis and radii. Native circle parameter conversion derives arc bounds from the
start direction and authored endpoint. A zero or full-turn arc is refused as
ambiguous. Native evaluation measures both line/arc endpoints against their
source coordinates under an explicit independent acceptance bound. Inconsistent
arc endpoints fail; they are not silently projected or moved to fit a circle.

The query uses the [shared analytic intersector](implementation-planar-curve-boundary.md).
It retains points, parameters on both operands, measured residuals, transitions,
empty results, and bounded or unbounded overlaps. Line parameters measure working
length along the unit native direction; they are not normalized to `[0,1]`.
Targets use a total order by entity ID. There is no first-hit selection,
coincidence merge, nearest-root choice, tessellation, geometry retry, or fallback.
A completed native result is not a certificate of root completeness.

Standalone and defining points are not curve operands. Construction curves and
centerlines still participate. All spline and offset-spline entities cause an
explicit preflight refusal, before any native curve is built. The current
Martensite v0.2.5 release lacks the construction extension now on kernel main;
[kernel issue #9](https://github.com/Oblikovati/RustyCADKernel/issues/9) remains
the owner release and application acceptance task. A missing spline must never
produce a successful query that omitted a possible crossing.

Errors retain the offending entity and original typed model or kernel-adapter
cause. The source chain still reaches native exceptions. Every target is resolved
before intersections start. A target failure returns an error without a partial
success report. Inputs, roles, constraints, and identities remain unchanged.
No API contract, persistence schema, release pin, or automatic constraint changes
are part of this query.

The 12 new integration tests check analytic line and conic roots, both arc
orientations, endpoint contacts, tangent contacts, unbounded opposite overlaps,
strict result bounds, source immutability, independent result ownership, scales
1e-6/1/1e6, native error chains, and preflight precedence over invalid geometry.
The established generic curve corpus remains the native evaluation evidence;
these checks add owned-sketch conversion and domain-policy evidence.

Line extension still requires certified endpoint/root selection, atomic endpoint
replacement with identity reservations, history and workspace ownership, typed
API replies, save/reopen, and native edit acceptance. Trim additionally requires
interval classification and survivor construction. Spline release consumption,
general profile adoption, and full R3 acceptance remain open. This query must not
be presented as completed trim or extension support.

## Local verification

`make gate` passed across all 13 application crates, all targets and features,
including native head and viewport tests on an isolated 1920×1080×24 Xvfb display
with software Vulkan and validation layer 1.4.321. Formatting, Clippy, build,
tests, doctests, coverage, Markdown, documentation links, SPDX, workflow checks,
and duplication passed. Every crate exceeded 80% line coverage. Adapter coverage
was 90.88% (2,142/2,357); this query boundary reached 95.30% (304/319).
Duplication was 0.75% (510/68,039 scanned lines).

After the gate, the conic test gained an explicit four-root count assertion so
that missing roots cannot pass a zipped-value comparison. All 12 query tests,
formatting, and all-target/all-feature adapter Clippy passed again. Production
source did not change after the gate. Documentation checks passed after this
verification record was added.

The conversion introduces no new numeric tolerance constant. Axis vectors,
zero line bounds, and one-turn limits express the native curve definitions;
acceptance bounds come from the caller. ID ordering is explicit. Linux is the
verified platform; Windows and macOS runtime acceptance remains open. No push,
PR, or publication is part of this local work.
