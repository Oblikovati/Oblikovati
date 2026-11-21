# Viewport Pan evidence baseline

All four scenarios and 32 steps passed in the native Linux desktop runner
on application commit `55f49da68cc289a68a96ba209d13dde8baa87550`, under Xvfb
and Vulkan Lavapipe. The runner restored the clean scenario snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`4303c8a64b9262841dba3b8b48a74e51581a6a0ef9d98798286d514c53155647`.

The desktop cases check the active toolbar control and status, Escape and
empty-view cancellation, Done in the context menu, and the switch to Zoom.
Two separate X11 input tests also passed. They check the camera after a drag
and verify that the Pan context menu owns the right click while Pan is active.
The capture shows the context menu, the active Pan control, and the visible
solid.

```sh
OBK_BDD_FEATURE=viewport-pan XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
XMODIFIERS=@im=none \
  tools/with-xvfb.sh cargo test --locked --offline -p oblikovati-head --lib \
  x11_pan -- --nocapture --test-threads=1
```

Both commands set `VK_DRIVER_FILES` to the installed Lavapipe manifest and
used separate temporary report directories. The three focused tests in the
[Go reference baseline](../2026-11-14-go-pan-baseline/README.md) passed again.
The Go tests cover gesture and camera behavior. The Rust tests check the
toolbar, status, and context menu.

The live feature names an Inventor Part. A native Part fixture is requested
in the live feature's `BLOCKERS.md`. The recorded results apply to the pinned
feature. Strict acceptance also requires build and runtime provenance in
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
