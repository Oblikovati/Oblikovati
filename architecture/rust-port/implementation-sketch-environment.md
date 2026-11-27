# Selected sketch edit and camera — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go sketch create, finish, restore and `scene.Camera.Facing` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The desktop owner can create and enter a sketch, enter an explicitly selected
existing sketch, and finish its edit. Entry requires the active part or assembly,
no open transaction and no current edit. Creation uses the existing canonical
command, solve and history path. Part creation can project the origin; assembly
creation does not project the part origin. Failed creation leaves no edit active.
Finishing cancels unfinished placement and clears the edit. It adds no history
entry; committed sketch commands retain their independent undo entries and the
normal scene refresh publishes their evaluated result.

A session continuity observation tracks the selected sketch at every published
history change. Ordinary edits and history restores preserve an existing sketch.
Removal, a lost placement or session retirement invalidates the observation
permanently. Undoing creation and redoing it in one host call batch cannot revive
an ended edit. Failed or vetoed changes do not invalidate it. Weak registrations
let retired observers be removed without retaining editors. The workspace also
exposes a focus identity: a successful switch away and back ends the edit even
before the next frame. Same-document activation, refused activation and closing
a background document preserve that identity. No wrapping counter is required.

The native window saves the return camera and faces the sketch plane before the
first pointer projection. It preserves view side, eye-target distance, scale and
projection limits, and uses the plane's Y axis for camera up. The pinned Go unit
standoff below its minimum camera distance is retained. Pan and zoom remain valid
while editing. Restoring the same sketch identity does not re-enter the camera
transition, matching Go history reattachment. Finish or lost edit context restores
the saved camera, including after resize. Entry failure clears the transient edit
and displays a named command notice; it preserves the created sketch and document.
Camera replacement updates the saved state only after the native camera accepts it.

Slot activation now uses this selected edit context. The old separate slot document
guard was removed. Script and native add-in owners forward the same edit state and
failure path. No public API contract, geometry algorithm or kernel source override
was added. Kernel/API pins remain v0.2.5/v0.158.0.

Native tests use the real owner, view, camera and renderer. They assert independent
camera coordinates, projection round trips, navigation, resize, capture/presentation
agreement, finish restoration and atomic failure. Session and owner tests cover
creation, document kinds, origin projection, nested entry, open transactions,
history continuity, focus continuity and wrapper forwarding. The pinned Go camera,
create, finish and history reattachment tests pass as an independent baseline.

This is an owner workflow and immediate camera transition. Native menus, ribbon and
browser selection, camera animation, edit-scope downstream feature rollback,
environment events, snapping and editable placement fields remain open. This step
does not establish complete sketch editing or general named solids. Native GUI
composition is now kernel application priority 3, recorded in kernel commit
`390cd0eb5`. Released spline/predicate prerequisites and general region-to-solid
support remain priorities 1 and 2. Source-only kernel work does not satisfy release
acceptance. All work stays local; Windows/macOS runtime acceptance remains open.

The cross-platform review checked fixed-width document identities, atomic lifetime
flags, non-reused focus identities, finite camera arithmetic and observed resize
extents. The camera's `1e-3` threshold and unit fallback are the pinned Go view
policy, not a modelling tolerance. The comment review checked history reattachment,
weak observer ownership and atomic camera replacement. Native foreground tests run
serially because concurrent windows can obscure the X11 pixels under test.

The final local `make gate` passed across all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests used Xvfb at
1920×1080×24 with software Vulkan and validation layer 1.4.321. Every crate exceeded
80% line coverage; head reached 95.67%, session 98.85% and viewport 93.84%.
New executable modules reached 96.18%–100%. Duplication was 0.83%
(595/72,086 scanned lines). The new match is the existing generic desktop owner
implementation bounds; no geometry algorithm was copied. The review removed the
repeated slot plane lookup. A regression test first demonstrated stale slot status
after a focus switch away and back, then passed with the shared live edit check.
