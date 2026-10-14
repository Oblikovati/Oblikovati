# Shared rectangle placement — partial R3/R4

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go rectangle recipes, tools, preview and placement input at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The model now supplies two-corner, centre and three-point rectangle recipes.
They reuse the existing rectangle seed constructor. Its directed-centre choice
retains the clicked corner direction, while the raw API still uses absolute
half-extents. The existing 21 independent Go seed fixtures continue to match
bit for bit. The duplicate placement coordinate calculation was removed. All
use four shared corner points and one closed boundary. Axis-aligned rectangles
retain two horizontal and two vertical relations. A centre rectangle also retains
both construction diagonals and a centre point constrained to one diagonal's
midpoint. A three-point rectangle uses the signed perpendicular depth of the
third point and three perpendicular relations; a fourth would be redundant.
The unconstrained ranks match Go: four, four and five degrees of freedom.

Width and height fields retain horizontal/vertical distance operands. Rotated
rectangles retain aligned length and width operands plus a steering angle, which
creates no dimension without a reference edge. The existing atomic placement
command owns points, members, relations and locked parameter sources. Invalid
indices or a late field failure publish no partial rectangle. Zero-length edges,
nonfinite inputs and overflow are refused before model adoption. No modelling
geometry is moved to make an input pass.

The old slot-only input state has been replaced by one placement state and an
explicit recipe choice. Slots and all three rectangles share ordered pointer
input, drag recognition, navigation suppression, projection, preview publication,
commit, restart, cancellation and history/focus invalidation. `start_slot_placement`
is a convenience entry into that same state. `start_sketch_placement` selects a
rectangle on the current sketch. Common inspection uses `placement_preview` and
`placement_active`; there is no second rectangle input engine.

Three-point placement accepts the base first, then waits for height. As in Go,
its complete recipe preview starts after the second point. Click-click and
click-drag-release use the same point path. An invalid base point or collapsed
height retains the current stage for correction. A successful commit creates one
history step and restarts the tool. Escape removes unfinished geometry. The
selected edit and native camera use the prior
[sketch environment step](implementation-sketch-environment.md).

Twelve independently generated Go fixtures cover all three rectangle forms in
four quadrants. The Rust checks compare point order, member roles, relation order,
field labels, values, witnesses and dimension orientation. Each fixture is applied
and solved, with matching rank and no redundant equations. The previous sixteen
slot fixtures remain checked. Regeneration produces identical JSON bytes. The
pinned Go recipe, rectangle tool and placement tests pass.

Model tests fix position and, for a rotated rectangle, orientation, then drive
width while height follows a parameter formula. All three forms remain at DOF zero
with no redundant equations through three sizes. Every corner and the centre are
checked against independent coordinates. Other tests cover recipe round trips,
undo/redo, invalid inputs, missing midpoint operands and late expression failure.
The coordinate assertions check achieved solve precision; a fixed point is a
solved equation and is not assumed to have bit-identical coordinates.

Owner tests compare click and drag results for each tool, history boundaries,
restart, invalid-stage correction and cancellation. Real native tests drive all
three forms through the owner loop, compare preview and committed points, verify
no preview mutation, clear the outline on commit, undo, and compare capture with
foreground X11 pixels. A captured centre rectangle was inspected visually: its
boundary is solid and its two construction diagonals are dashed. The same native
renderer and sampler used by slots produce these outlines.

The later [committed sketch display step](implementation-sketch-presentation.md)
keeps geometry visible after commit and Finish. Native menu/ribbon selection, hit
testing, editable field widgets, snapping, annotation positions and the other
placement tools remain open. These mouse tools do not
automatically fix their position or add dimensions; the model's locked-field path
is verified separately. Full GUI composition and general region-to-solid support
remain kernel release blockers. This step does not establish R3 or R4 completion.
Kernel/API pins remain v0.2.5/v0.158.0 with no local source override. Work is local
only; Windows/macOS runtime acceptance remains open.

The cross-platform review checked finite binary64 placement arithmetic, signed
pointer subtraction, the existing four-framebuffer-pixel drag threshold, and local
recipe indices that never become persisted identities. Recipe order is explicit;
there is no map-dependent point or relation ordering. The comment review retained
the Go rationale for independent perpendicular constraints and reciprocal/multiply
seed order. Shared distance-field resolution and field construction avoid separate
implementations for rectangles and slots.

The final local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used a 1920×1080×24
Xvfb display, software Vulkan and validation layer 1.4.321. Every crate exceeded
80% line coverage. Model reached 98.82% and head 95.81%; the rectangle recipe,
shared seed, field helpers and placement state/tool modules reached 100%. The
owner input module reached 98.56%. Duplication was 0.82% (595/72,369 scanned lines),
with no additional duplicated lines or clone pairs relative to the wave base.
