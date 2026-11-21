# Owned regular polygon creation — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/composite.go`, `AddPolygon` and `polygonVertices`,
and `addin/router/sketch_add_entity.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The typed `sketch.addEntity` call accepts `kind: "polygon"`, a center, a through
point, and at least three sides. The `circumscribed` variant treats the through
point as an apothem endpoint; other variants use an inscribed polygon, as in Go.
The common coordinate resolver applies working units and expression precedence.
Seed expressions do not become live parameter bindings.

The model creates one closed chain through the shared polyline builder, then a
fresh center and a construction circle. The circle radius is measured from the
first authored vertex, as in Go. Every vertex receives a point-on-circle relation;
each consecutive pair of lines receives an equal-length relation. There are
`2*n - 1` regularity constraints. Their IDs belong to the sketch-local constraint
namespace; points and curves use document-global entity reservations.

The requested construction flag applies to the polygon lines. The defining
circle is always construction geometry. The reply contains the first line,
every polygon line, and all vertex IDs followed by the center ID. The defining
circle remains available through ordinary entity queries. It is not inserted into
the reply's polygon line array. No inference, centerline, or closure options
change this composite. The API contract and persistence schema stay unchanged.

Creation edits a private geometry copy. It validates the seed and defining circle
before reserving identities. Failed allocation of a line, circle, or constraint
cannot publish a partial polygon. IDs already reserved remain retired after
failure, abort, observer veto, undo, and save/reopen. The shared command boundary
runs the solver and prepares the reply before adoption.

These are authored analytic sketch entities with regularity relations, not a
certified profile. Invalid counts, nonfinite coordinates, zero or unrepresentable
radius, and unallocatable vertex counts return errors. Finite values outside the
arithmetic's representable range can be refused. No epsilon, alternate topology
classifier, perturbation, or kernel fallback is added. Go and Rust trigonometric
libraries can round seed coordinates differently; no bitwise parity is claimed.

Twenty independent fixtures call the pinned Go implementation for inscribed and
circumscribed polygons with 3, 4, 5, 8, and 17 sides, with translated and rotated
inputs. Rust consumes committed JSON from
`crates/model/tests/polygon_seed/go_oracle.go`. The comparison uses a stated
binary64 rounding bound; this bound is not a topology tolerance. Model tests check
ordered owned identities, exact constraint references, recipe round trips,
nonfinite and extreme input refusal, and atomic entity/constraint exhaustion.
A five-sided polygon reaches zero degrees of freedom after constraining its
center, radius, and orientation, then recomputes after a radius parameter edit.

Typed API tests check both variants and Go's default, real line/point/center IDs,
construction-circle ownership, current reference keys, count and radius errors,
undo/redo, save/reopen, and entity plus constraint retirement after abort/veto.
The native test drives the real document owner through the typed API. Four free
polygon variables make the stored prism unavailable; scene publication removes
that body, and undo restores the exact original image. The test checks the five
new curve rows and their references. It does not prove polygon overlay rendering
or extrusion of the new profile.

The application still consumes Martensite v0.2.5 and API v0.158.0. Interactive
polygon tools, general sketch-region construction, and full R3 acceptance remain
open. This work does not change the kernel priority queue or release pins.

The full local `make gate` passed for all 13 application crates, all targets and
features. It includes format, Clippy, build, tests, documentation tests, coverage,
Markdown/link checks, SPDX, workflow lint, and duplication. Native tests ran
serially on a 1920×1080×24 Xvfb display with software Vulkan and validation layer
1.4.321. The polygon native test passed. All crates exceeded 80% line coverage.
The seed module, polygon history, relations, typed errors, and API input module
reached 100% (47/47, 58/58, 31/31, 12/12, and 27/27 lines). Model coverage was
98.80%; automation was 96.31%. Duplication was 0.77% (527/68,786 scanned lines).
The new ten-line match is typed error-cause forwarding shared in form with line
creation; polygon allocation, chain construction, and solving are not duplicated.

The Go polygon tests pass, and fixture regeneration is byte-identical. The
cross-platform and comment review found no new production epsilon, persisted
platform-sized identity, unordered allocation, or removed intent comment. Side
counts are converted from the wire integer with an explicit representability
check. Windows/macOS runtime acceptance remains open. No push or release is part
of this local change.
