# Owned sketch polyline creation — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Go reference: `model/sketch/composite.go`, `AddPolyline` and `closedLoopPoints`,
and `addin/router/sketch_add_entity.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The existing API `sketch.addEntity` accepts `kind: "polyline"`. An open chain needs
at least two positions; a closed chain needs at least three. All positions become
new hidden defining points, allocated in input order before any line. Lines then
connect adjacent point identities in that order. A closed chain adds a final line
from the last point to the first. Numeric coincidence does not merge separately
authored vertices. Repeated positions retain the Go seed behavior; creation does
not certify a nondegenerate boundary or a valid profile.

The response contains the first line as `entityID`, every line in `entityIDs`, and
all vertex IDs in `pointIDs`. Point IDs are unique and ordered by first use. Every
line receives the requested construction flag. As in Go's composite route,
`centerline` and `variant` are ignored, and no inference constraints are added.
Entity queries, reference keys, copy namespaces, and saved recipes use the existing
line and defining-point representation. No API DTO, wire method, or schema changes.

The existing point-input path handles centimetre literals and length expressions.
Nonempty `pointExprs` take precedence over `points`; expressions provide seed
coordinates and do not create live parameter bindings. The owner is validated
before mutation. Count and finite-coordinate validation precede reservations.
The shared primitive point and line builders replace the need for a second
allocator or geometry-adoption path. Geometry is published only after the entire
chain is built. IDs remain consumed after allocation failure, reply rejection,
veto, undo, or abort, and remain reserved after save/reopen.

Tests cover open and closed connectivity, stable allocation order, hidden points,
construction flags, ignored composite fields, repeated positions, invalid counts,
non-finite coordinates, missing owners, and exhaustion at every allocation step.
Session and typed API tests cover reply rejection, working units, expression
precedence, current reference resolution, part and assembly owners, read-only
refusal, undo/redo, abort, veto, save/reopen, and copied reference namespaces.

The native test adds a closed three-vertex chain to the stored parameter-driven
prism through the typed API. It checks all returned point and line IDs, every line's
coordinates and reference resolution, and retention of all original sketch rows.
The six new free coordinates make the whole-sketch readiness check refuse the
prism. The scene removes that unavailable body. Undo restores the byte-identical
original rendered image; the existing X11 capture check verifies presentation.
This is not evidence of a rendered sketch-chain overlay or automatic extrusion
of the new boundary. Those consumers remain separate work.

The Go `TestAddPolyline` tests pass independently. Their closed-profile assertion
checks the Go reference, not the still-open Rust region-construction contract.
Interactive polyline authoring, general profile regions, and full R3 acceptance
remain open. The app retains its exact Martensite v0.2.5 and API v0.158.0 pins.

The full local `make gate` passed for all 13 application crates, all targets and
features. It includes formatting, Clippy, build, tests, documentation tests,
coverage, Markdown/link checks, SPDX checks, workflow lint, and duplication.
Native tests ran serially on a 1920×1080×24 Xvfb display with software Vulkan
and validation layer 1.4.321. Every crate exceeded 80% line coverage. The new
polyline model, session command, workspace forwarding, and API reply modules
all reached 100% line coverage (84/84, 18/18, 16/16, and 12/12 respectively).
Overall model coverage was 98.79%, session 99.06%, and automation 96.27%.
Duplication was 0.76% (517/68,375 scanned Rust lines); the new seven-line match
is the shared signature of the session and workspace forwarding methods.
Point reservation and curve adoption continue to have one implementation.

Cross-platform and comment review found no new tolerance literal, platform-sized
permanent identity, unordered allocation, or numerical geometry construction.
Existing primitive-creation comments remain intact. Windows and macOS runtime
acceptance remains open. No push, PR, or release is part of this change.
