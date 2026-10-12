# Native viewport navigation — partial R4/R7

Date: 2026-10-12. Tracking: application
[#3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Kernel: Martensite v0.2.1, `193572dc3665598c4948777c87375bad7dc8114a`.
Go references: `head/ui/navigate.go` and `scene/camera.go` at the
[fixed source revision](source-map.md).

## Implemented path

X11 copies button, motion, wheel, and navigation-key events into owned Rust
values. Key lookup first checks that the event uses the window's display.
Malformed null or foreign display pointers are rejected before Xlib can
use them. The regression test covers the null-display synthetic key event
that crashed the first combined native run.

The desktop controller retains the Go default bindings and sensitivities:

- Middle-button drag pans. Shift+middle-button drag orbits. Shift wins when
  Control is also held; Control+middle retains the default pan binding.
- Wheel zoom uses `0.9` per notch and keeps the cursor's target-plane point
  fixed. F2/F3/F4 enable left-drag pan, zoom, and orbit respectively.
- Drag zoom uses `1.01` per vertical pixel. Orbit uses `0.01` radians per
  pixel, yaw followed by pitch about the current target.
- Plain left and right drags do not navigate. Matching button release ends
  capture. Focus loss, unmapping, closing, or Escape clears held navigation.
  Captured motion can continue outside the window. Hover motion does not
  request another frame.

Commands reach the existing Martensite view; the application does not add a
camera rotation or projection engine. Its Rotate arguments follow OCCT's
`-Ax` about screen up and `Ay` about screen right. A candidate kernel camera
is validated before adoption. Rejected input restores the previous camera,
completed capture, and application camera generation. A successful command
invalidates the old capture and requests a frame in the normal owner loop.

Navigation changes camera state only. The current scene generation and
analytic model results remain unchanged. It does not enter document undo or
recompute history. No native event, camera handle, or display pointer escapes
the platform/renderer boundaries.

## Validation

Pure controller tests cover default and modal bindings, modifier priority,
release, cancellation, focus/unmap/close, captured motion, wheel factors,
stationary input, and overflow refusal without moving the drag anchor.
X11 tests use the real keymap and send wheel/motion events through the native
queue. They also check invalid display pointers and auxiliary button events.

Native Vulkan tests check analytic pan/orbit camera values, camera and scene
generations, cursor-anchored zoom round trips, rejection and rollback, retirement,
and equality between captured pixels and the X11 drawable. A stored part opened
through the typed client changes its displayed pixels after navigation, requests
a frame, and produces no new model evaluation. Tests run on this Linux host.

```sh
CARGO_NET_OFFLINE=true cargo test --locked -p oblikovati-viewport -p oblikovati-head
CARGO_NET_OFFLINE=true make gate
```

The full local Linux gate passed for all twelve application crates, including
native display tests, per-crate coverage, clippy, formatting, doctests,
documentation, workflow validation, SPDX, and duplication checks. Duplication
was 0.58%. No remote test, push, or PR was performed.

## Remaining work

This implements core navigation for the current orthographic native view. It
does not close R4/R7 or claim the full Go camera/UI surface. Configurable
bindings, wheel inversion, constrained orbit, the orbit ring and ViewCube,
window zoom tools, navigation bars, animation, perspective views, saved
per-document views, and public camera automation remain open. Input capture
by ImGui panels and active modeling tools must be integrated before complete
UI acceptance. Windows and macOS remain deferred.

## Home and fit continuation

`F6` now runs Home View on the native owner thread. It clears an active drag,
restores the Go Y-up isometric direction (eye offset `+X,+Y,+Z`), and fits
displayed geometry through released Martensite `V3dView`. Key release does
not repeat the command. The viewport also accepts `Fit { margin }` for future
Zoom All controls, retaining the current orientation. No ribbon control is
claimed. Both commands require a finite margin in `[0,1)`.

The current view is orthographic. Fit preserves its eye-to-target distance
for all inputs while the kernel computes the center and scale, then computes
the clipping range with `ZFitAll`. This is a camera invariant, not a geometry
repair: released revision `193572dc` sets fit distance to displayed box depth,
which becomes zero for a flat shape viewed straight on. The regression first
failed with coincident eye/target and now restores visible native pixels.
No kernel source or document geometry is changed. Perspective fit remains
open and cannot assume this orthographic distance rule.

Empty Fit preserves its completed capture. Empty Home changes orientation
around the current target and retains scale and clipping range, as Go
`scene.Camera.Home` does. A command that leaves the camera unchanged does
not increment camera generation or discard a completed capture. Invalid
framing input is refused before any camera change. Native tests recover a
panned-away triangle and a stored prism, verify direction and rendered pixels,
and prove that the scene generation and retained document evaluation do not
change. Native X11 tests also translate the actual F6 keymap.

## Previous View continuation

`F5` restores the latest camera recorded before a successful discrete Home
or Fit command. Each viewport retains at most 64 camera values, as Go
`app/view_history.go` requires. Continuous pan, orbit, and zoom do not add
entries. A failed command does not add an entry; a failed restore retains
the entry for retry. A successful restore removes it, including when the
saved camera already equals the current camera. Empty history is a no-op.

History retains owned kernel camera members without native handles or viewport
pixel dimensions. Rebuilding from eye/target alone introduced target rounding
in the history-order regression; copying the members restores the exact stored
camera values. Restoration
uses the current drawable extent. An unchanged camera preserves its completed
capture; a changed camera invalidates only camera generation. Tests check
retention order and the bound, independent viewports, failure recovery,
native F5 press/release, restoration of a stored prism's exact rendered
pixels, and restoration after native resize. The full local Linux gate is
the acceptance command; this does not close the wider R4/R7 requirements.

The native suite had one pan assertion failure during the first combined run
(`-5` rather than the fixed expected `-2.5`). The same suite then passed and
eight further concurrent native-suite runs passed (40 tests). Those repeats
do not prove the cause of that one result. The existing assertion remains
active; no test serialization or weakened acceptance was added.

## Native size oracle correction

A later repeat trace identified a failed numeric assertion's input: the
zoom window requested `128×96`, but its completed capture reported `320×240`.
At pointer `[96,24]`, scale 20, and zoom factor 0.9, the target shift was
`[-0.5333333333333332, 0.7999999999999998, 0]`, which is the expected projection
for that drawable. The fixed `128×96` expectation was an invalid premise.
This evidence does not identify the source of the native size change.

Native tests now check their own X11 title and use the completed drawable's
dimensions in the independent world-coordinate oracle. Exact fixed-size pan
and anchored-zoom assertions remain on an offscreen target, where no window
manager can change those dimensions. Native image round trips, scene and
camera generations, and X11 presented-pixel checks remain active. No renderer
code, validation setting, or test concurrency was changed for this correction.
