# Home Recent Documents resize

All 37 amended Home Recent Documents scenarios passed on Wayland, for
408 steps. The resize scenario also passed all five steps on X11.

The resize scenario opens Recent with 24 copies of the published Part2
fixture and varied timestamps. It verifies full timestamps at the
initial width, then drags the native window edge to about 880 × 540.
Both date columns switch to complete date-only values and the visible
file names stay readable. After native wheel scrolling, the last file
name clears the document tab strip by at least 12 pixels. The date cells
retain their full timestamps on hover.

The native build used application commit `0907906c8` and existing local
GUI filesystem edits. Wayland ran under nested Weston with Vulkan
Lavapipe. X11 ran under Xvfb. The scenario and support files matched the
recorded input hashes before and after the runs. Strict build and runtime
provenance remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 resize report](x11-resize.json)
- [Published input hashes](inputs.sha256)
