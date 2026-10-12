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
fit/window zoom tools, navigation bars, animation, perspective views, saved
per-document views, and public camera automation remain open. Input capture
by ImGui panels and active modeling tools must be integrated before complete
UI acceptance. Windows and macOS remain deferred.
