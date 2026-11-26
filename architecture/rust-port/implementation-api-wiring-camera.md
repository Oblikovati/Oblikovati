# Native camera API wiring

The Rust desktop handles API 0.166.0 `view.getCamera` and
`view.setCamera` on the native window owner. It resolves the active or
explicitly addressed open document and uses that document's selected
window slot. A background write changes the saved camera for the
addressed slot. It leaves the visible window on its current camera.
Switching tabs restores each window's camera and projection.

The look-at frame uses a vertical field of view in radians. The host
converts that angle to the viewport height at the target and retains
the exact requested wire frame while the renderer's pose and scale
match. A later navigation change reports the actual viewport frame.
The setter validates the frame before moving the camera. Omitting
projection preserves the view's mode, including the hybrid mode with
orthographic face views.

## Verification

Four camera tests covered the native background slot, its rendered
camera after activation, validation, and document addressing. Nine
desktop-owner tests covered document and window selection. The live
Wayland MCP driver round-tripped the active camera through the
installed Bridge and rejected a coincident eye and target when using
the temporary Bridge build described below.

The checked-out Bridge at `3bb0352` advertises a no-argument
`get_camera` tool and omits `projection` from its `set_camera`
schema. A temporary Bridge build added those two fields to exercise
the full wire contract without changing its repository. That run
passed explicit background reads and writes, camera restoration on
activation, projection retention, and rejection of an unknown
document and a degenerate frame. The Bridge schema changes remain open.

A five-second client stress run around a 50 mm cube requested 60 Hz
through the installed Bridge. It completed 113 camera commands with
no transport or host errors. It achieved 22.6 updates per second at
43.841 ms median and 54.234 ms p95 round-trip latency. Host trace data
was unavailable. These are MCP request measurements.
