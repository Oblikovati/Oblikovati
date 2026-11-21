# Slot placement input on the owner loop — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go `app/sketch_placement.go`, `sketch_create_tools.go` and
`sketch_preview.go` at `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The desktop owner can activate straight-slot placement on an explicitly supplied
active document and sketch. Width is a positive finite length in model working
units. Activation resolves the current sketch plane and refuses an open document
transaction. It does not silently select the first sketch or create a new sketch.

The owner loop accepts click-click and press-drag-release through the same point
collection path. Strictly more than four framebuffer pixels marks a drag, matching
Go #2014. A release can be the first report of movement. A click release does not
create a second point. After a successful commit the tool waits for a new first
point; the trailing release cannot start another slot.

Point collection and the current cursor recipe are transient owner state. They
allocate no document IDs and create no history entry. The cursor and commit use
the same constrained slot recipe builder from
[slot placement recipes](implementation-slot-placement-recipes.md). A valid second
point commits through the existing atomic session command. No new geometry,
constraint solver or allocation path is added. Typed field entry is not connected
in this step; this input path supplies no locked dimensions.

Escape, focus loss, unmapping and close cancel the tool. Navigation gestures clear
the pending press so their release cannot create geometry. An edge-on plane clears
the cursor recipe. Projection or invalid-seed errors reach native command feedback
and leave the first point available for retry, with the press cleared. The native
status text asks for the first or second centre point.

The tool stores its document generation. An external edit, undo/redo, active-document
change, close or open transaction invalidates its context before the next input.
Read-only preview queries also refuse an invalid context before the next frame.
A successful tool commit advances its own generation before repeated placement.

Navigation is now applied once per event, immediately before owner processing of
that event. The old batch navigation/expose wrapper was removed. This prevents a
later zoom in one event batch from changing the camera used for an earlier click.
The frame scheduler uses the returned view-change flag to request presentation.
Script and native add-in owners forward this input path.

Owner tests cover equivalent click/drag geometry, threshold/release behavior,
repeated placement, cancellation, navigation, invalid input, unavailable projection,
external edits, transaction refusal and script forwarding. The pinned Go placement
tests pass independently. Native tests use a named scripted event source with the
real native camera, renderer, owner loop and document. They compare the pending
recipe with the committed points, verify no model change while collecting points,
check native prompts and undo against the original viewport pixels, and compare
click/drag document recipes. A click/zoom/click batch verifies camera event order.

Activation is currently an owner entry point, not a menu or toolbar command in the
executable. The [native outline step](implementation-slot-preview.md) now draws
the current recipe. Selection of the active sketch in the UI, editable field
widgets, inference/snapping, and the other placement tools remain open.
These tests do not establish full interactive sketch editing or general solids.
The kernel profile-to-solid blocker and its priority remain unchanged.

Kernel/API pins remain v0.2.5/v0.158.0. There is no local kernel override, push or
release. Windows/macOS native runtime verification remains open.

The full local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests ran on a
1920×1080×24 Xvfb display with software Vulkan and validation layer 1.4.321.
Every crate exceeded 80% line coverage. Placement state reached 100%, owner
integration 97.93%, and the head crate 95.40%. Duplication was 0.79%
(562/71,075 scanned lines); the additional match is owner generic bounds,
with longer existing script/add-in forwarding. No input or geometry algorithm
was copied.

The cross-platform and comment review checked fixed-width identities, signed pixel
subtraction, the documented four-pixel threshold, transaction boundaries and native
input order. Pixel subtraction converts each `i32` to `f64` before subtracting,
so extreme pointer values cannot overflow an integer delta. Temporary recipe
indices never become persisted IDs. The old batch navigation wrapper was deleted
when the ordered event path replaced it, and its native camera tests remain in
the gate.

Selected sketch ownership and camera entry/exit are covered by the later
[sketch edit step](implementation-sketch-environment.md).

The later [rectangle placement step](implementation-rectangle-placement.md)
replaces the slot-only input state with one shared placement pipeline.
