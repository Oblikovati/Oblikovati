# Sketch Fillet from the Part ribbon

The X11 regression opened Fillet in the Part's only planar sketch. Escape
after the first line pick preserved the original geometry. Picking two
adjacent lines created a tangent arc at the initial 5 mm radius. Ctrl+Z
restored the original sketch and Ctrl+Y restored the rounded corner. The
test closed its window. See the [framebuffer capture](sketch-fillet-from-ribbon.png).

The reference at Go `aa8297d1b` uses two line picks and starts with a 0.5 cm
radius. The previous [Go ribbon baseline](../2026-11-14-go-sketch-ribbon-baseline/README.md)
records the different out-of-edit ribbon behavior. The Rust ribbon calls the
typed Sketch API for the geometry change. Three focused API fillet tests passed,
including history, reopen, and rejection of an oversized radius.

Both native feature runs used Linux X11 with Xvfb and Vulkan Lavapipe. The
pinned source passed one of three scenarios and 21 of 23 executed steps.
The live source passed three of five scenarios and 31 of 33 executed steps.
The two failed scenarios both passed the Fillet availability check and stopped
at Sketch Text. The checks after those failed steps still need a passing run.

- Pinned feature SHA-256 `59959ff37cf45000a2fc634c665c70a0b784ae721f6fe8db1dd7d4399d659a0d`.
- Live feature SHA-256 `b3be8114d042b3b5d7c160569262657afed7c6e167711c2e6abc485bdc6ee9c9`.

These results show the current behavior. The three pinned R0 baseline records
describe the expected behavior and test scope. Strict scenario verification
still requires bounded build and runtime inputs under #3978.
