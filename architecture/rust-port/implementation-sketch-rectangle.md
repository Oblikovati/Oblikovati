# Owned sketch rectangle creation — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/composite.go` and
`addin/router/sketch_add_entity.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The typed `sketch.addEntity` call accepts `kind: "rectangle"` through the existing
API contract. The `center` variant takes a center and corner. The `threePoint`
variant takes the first edge and a depth point. Other variant names select the
opposite-corner constructor, as in Go. Counts must be exactly two or three.
The third point is projected onto the first edge's perpendicular to author the
three-point seed. The arithmetic operation order follows the Go implementation.

All constructors feed the existing closed-polyline operation. Four fresh hidden
points precede four lines in allocation order. Each line shares endpoint identity
with its neighbours; the last line returns to the first point. The reply contains
the first line, all four lines, and the four vertex IDs. Construction applies to
every line. The `closed` and `centerline` request fields do not change this
composite, and no inference or geometric constraints are added, as in Go.
The shared chain reply replaces the polyline-only reply implementation.

These are authored seed coordinates, not proof of a valid profile or a fully
constrained rectangle. Coincident corners and zero depth retain Go's draft seed
behavior. The three-point first edge must have a finite, nonzero representable
norm under Go's expression. Nonfinite inputs, intermediate coordinates, and
outputs fail before any identity reservation. Extreme finite inputs can therefore
be refused. There is no epsilon, coordinate perturbation, alternative normalizer,
B-rep operation, or new profile classifier in the application.

Point expressions use the common unit resolver and take precedence over literals.
They supply initial values, not live parameter bindings. The existing transaction
boundary handles solve, undo/redo, abort, observer veto, and recipe persistence.
A later solve can change the seed coordinates. General profile construction and
parameter-driven extrusion remain separate acceptance requirements.

The model tests compare all vertex coordinate bits for 21 independent Go cases:
all three constructors, reversed and translated edges, positive and negative
depth, collinear zero depth, and large and small coordinates. The committed
`crates/model/tests/rectangle_seed/go_oracle.go` calls the Go sketch constructors;
Rust tests consume its static JSON without Go. Replaying the capture produced
identical bytes. Separate cases check signed zero, repeated corners, nonfinite
positions, and unrepresentable arithmetic. Go's rectangle tests also pass.

Typed API tests cover all variants, Go's default variant behavior, ordered shared
vertices, construction flags, ignored centerline, no implicit constraints, count
and first-edge errors without reservations, units, expression precedence,
undo/redo, save/reopen, current keys, and persisted ID retirement after abort/veto.
Existing polyline tests cover the shared allocation and reply path.

The native rectangle test uses the actual document owner and typed API. It checks
all four lines, coordinates, reference resolution, and retained original rows.
Eight new free coordinates make the sketch's prism unavailable; publication must
remove that body. Undo must restore the exact original image. The shared native
chain harness also retains the polyline case. This test does not establish a
rendered sketch overlay or extrusion of the new rectangle.

The Martensite v0.2.5 and API v0.158.0 pins and the saved recipe schema are
unchanged. Interactive authoring and the full R3 editable-part gate remain open.

The full local `make gate` passed for all 13 application crates, all targets and
features. This includes format, Clippy, build, tests, documentation tests,
coverage, Markdown/link checks, SPDX, workflow lint, and duplication. Native tests
ran serially on a 1920×1080×24 Xvfb display with software Vulkan and validation
layer 1.4.321. Both native chain cases passed. Every crate exceeded 80% line
coverage. Rectangle seeds reached 61/61 lines, chain inputs 35/35, and chain
replies 13/13 (all 100%). Model coverage was 98.79%; automation was 96.29%.
Duplication was 0.75% (517/68,509 scanned lines, 61 clones), with no new clone.

The cross-platform and comment review found no new tolerance constants,
platform-sized persisted identities, unordered allocation, or removed intent
comments. Seed arithmetic uses binary64 and keeps the Go expression order;
Windows and macOS runtime verification remains open. This change is local only.
