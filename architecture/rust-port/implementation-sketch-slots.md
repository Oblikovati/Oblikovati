# Owned sketch slots — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go `model/sketch/composite.go` and `addin/router/sketch_add_entity.go`
at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The existing typed `sketch.addEntity` call now creates slots with all six baseline
placements: center-to-center (default), `overall`, `slotCenter`, `arc`,
`arcCenterPoint`, and `arcThreePoint`. Unknown variants retain Go's default
center-to-center behavior. Point expressions take precedence over literals;
width is a length expression and the center-point arc uses `endAngle` as a
signed angle expression. Inputs are seeds, not new live dimension bindings.

All placements feed one owned slot graph and one atomic adoption path. Straight
slots have six distinct defining points, two lines, and two clockwise cap arcs.
Curved slots have eight defining points and four arcs, including separate owned
inner and outer centers at the same seed position. Adjacent members share endpoint
IDs. The boundary order and winding match the pinned Go constructors. Construction
role applies to all four members. Only the existing arc circularity relations
are added; no hidden tangency, equal-radius, width, or center-coincidence design
constraints are invented.

The model reuses the existing primitive builder. Point allocation was extracted
from the polyline builder into a shared helper; the old allocation loop was
removed. Points are reserved before boundary members, in the reference order.
A model failure cannot adopt part of a slot. Session failures, reply refusal,
abort and undo retain the existing rule that allocated identities are retired.
The wire reply has four boundary IDs. Straight slots expose their four corner
point IDs; curved slots return null point IDs, matching Go's `pointBearer`
behavior. Hidden centers remain model-owned and persist in the native recipe.

The direct placements are application authoring formulas, not a kernel offset,
blend, or topology engine. The three-point placement uses the injected released
Martensite circumcircle service. After the normal automatic solve, its callback
checks the actual two concentric-center candidates and actual cap centers against
the original three centerline inputs. Verification uses the explicit physical
policy from [three-point curves](implementation-three-point-curves.md). A refusal
rejects the complete slot before adoption. There is no local kernel source
override, second circumcircle engine, tolerance widening or coordinate nudge.

Widths must be positive and finite, with a representable half-width. Coincident
centers, unrepresentable directions/coordinates, an overall length no greater
than width, a nonpositive inner radius, and collapsed caps are explicit failures.
These are representability and valid-seed guards. Go can accept some invalid
widths or collapsed results; those outcomes are not evidence of valid slot
functionality. No new absolute geometric epsilon is used in production.

Twenty independent fixtures capture all five direct placements from the pinned
Go source, with two widths and both sweep directions. Tests compare seed positions
with a stated binary64 rounding bound and compare every center/endpoint index and
winding exactly. Regenerating the Go capture produces identical JSON bytes.
Further tests cover invalid quantities, malformed point counts, unit conversion,
expression precedence, model recipe round trips, reference keys, undo/redo,
save/reopen, and allocation failure at every point/member position. A named fake
construction service proves rollback after final verification refuses the slot.

The native desktop test drives all six placements through the typed API and the
real kernel service. It verifies four new boundary entities and resolvable keys,
observes the free profile's unavailable-feature state, then checks that undo
restores the original viewport pixels. Straight slots add ten free degrees of
freedom and curved slots add twelve; this agrees with their owned points and
structural circularity. No fully constrained slot or general valid solid is
claimed by this test. General analytic profile-to-solid construction remains a
kernel blocker, and full R3 remains open.

The kernel and API release pins remain v0.2.5 and v0.158.0. Windows/macOS runtime
checks remain open. This work is local only; no push or release is included.

Go also has separate GUI placement recipes in `model/sketch/recipe_slot.go`.
Those include construction centerlines, tangency/equal-radius relations, and
interactive length/angle/width fields. They are not the constructors called by
`sketch.addEntity`. Their Rust recipe and interactive placement port remains
required; this API step does not complete that separate user flow. The six
pinned Go composite-constructor/profile tests pass independently.

The complete local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests ran serially on a
1920×1080×24 Xvfb display with software Vulkan and validation layer 1.4.321.
Every crate exceeded 80% line coverage. Slot input routing, command integration,
owned creation, and both placement modules reached 100%; shared seed validation
reached 93.15%. Model coverage was 98.79% and automation 96.48%. Duplication was
0.77% (539/69,879 scanned lines), with no new detected clone.

The cross-platform and comment review found no new absolute tolerance, fallback,
map-ordered allocation, or platform-sized persisted identity. Local array indices
refer to the fixed seed graph; permanent identities remain `u64`. The Go fixture
uses a map only for point lookup, never result iteration. The shared allocation
extraction preserves the polyline operation order, and the full gate includes its
existing identity, exhaustion, solve, native, and history regression tests.
