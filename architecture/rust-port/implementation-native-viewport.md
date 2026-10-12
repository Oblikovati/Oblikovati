# Application-owned Linux viewport — partial R1V

Tracking: [#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).
The accepted boundary remains [ADR-0008](../decisions/ADR-0008-vulkan-renderer.md).
This document records implementation evidence. R1V remains open.

## Released dependency and ownership

`crates/viewport` consumes Martensite v0.1.2 at
`19108f6650cdfca7050febc759d91bdfec001769`, through the release pins in
`Cargo.toml` and `Cargo.lock`. It does not use the active sibling kernel checkout.
The application owns X11 windows, bounded lifecycle event polling, and view,
scene, camera, and surface generations. Martensite owns the Vulkan driver,
retained structures, cameras, swapchains, submission, and GPU captures.

A runtime and its views stay on the owner thread. Compile-fail tests reject
sending either type to a worker. Each view retains its runtime. View removal
precedes native window destruction. Closing a view twice is safe. Offscreen
capture is an explicit target choice. A hidden native window refuses redraw;
it cannot report offscreen rendering as successful native presentation.

## Frame validity and input checks

Scene and camera replacement validate inputs before replacing the current
presentation. A failed draw invalidates capture. Capture metadata identifies
the view and completed scene, camera, and surface generations. Capture checks
the current drawable size, including a size change that the owner has not yet
polled. A size change during redraw invalidates the frame and requires new surface input.
Generation exhaustion is checked before an explicit resize changes the window.
Hide and remap invalidate completed frames and advance the surface generation.

Camera setup uses the kernel's combined eye-and-center setter. The released
`set_center` method can ignore a target change at the same distance. A regression
test verifies the target at that boundary. Nonfinite values, unusable direction
magnitudes, parallel viewing/up directions, and invalid clipping ranges are
refused. Native image capture accepts disjoint contiguous RGB masks and releases
its XImage through an owned guard, including failure paths.

## Local Linux evidence

Run the viewport checks with the existing X11 display and Vulkan device:

```sh
cargo test --locked -p oblikovati-viewport
```

The native test presents a red triangle on a blue background. It compares every
actual X11 drawable pixel with the completed GPU capture after initial display,
resize, camera replacement, remap, and creation of a second view. Removing the
first view must leave the second view usable. Kernel Vulkan validation errors
must remain empty. Separate recording-device tests exercise failed draws,
source replacement, stale captures, size changes, and exhausted generations.
Native event tests verify focus, exposure, resize, target identity, and the
format and atom checks for a close request.

The checked host uses `AMD Radeon 8060S Graphics (RADV STRIX_HALO)` on display
`:1`. Validation runs stay on this Linux host. No remote push or CI run is part
of this work. The full local command remains `make gate`; workspace discovery
includes the new display-dependent crate.

## Remaining port work

This adapter currently accepts presentation triangles and an orthographic
camera. It does not yet connect to modeled B-reps, picking, the application event
loop, or a native editor panel. Tessellation remains a derived view and cannot
become a modeling oracle. Kernel capacity beyond OCCT still belongs in
Martensite.

Shared Dear ImGui composition still needs the kernel-owned interface described
in [the prior viewport evidence](implementation-viewport.md). Application
integration must consume that interface without adding a second Vulkan renderer.
These remaining checks prevent a claim of complete R1V or complete application
parity. Windows and macOS work remains deferred.

## Idle event wait

The native target can now wait on its X11 connection and a `ViewportWake` event
counter together. Worker clones contain no model or window handles. Wakes are
nonblocking and coalesced; a wake sent before the owner starts waiting remains
pending. Events already buffered by Xlib return immediately and remain available
for bounded polling. Interrupted waits return to the owner loop. Broken event
sources and invalid durations produce named errors. An offscreen or closed target
cannot masquerade as a native event source. This supplies the legacy #1493 idle
wake behavior without a second renderer or a periodic idle GPU redraw.

## Resize during presentation

A full local gate exposed a window size change during rendering. The old native
test assumed that every presentation would complete without a concurrent size
change. A recording backend now changes its target during `render`, reproducing
the same public-call failure deterministically without window-manager timing.

`Viewport::present` returns either a completed current capture or the measured
`SurfaceChanged` extent. The owner can schedule a new frame for that new input;
it cannot return the old image as current. The strict `redraw` convenience still
refuses an interrupted frame. Renderer faults and incorrect backend capture
sizes remain errors. They cannot enter the surface-change path. The native test
uses a bounded surface-event loop and still requires exact displayed pixels for
every completed frame. This is handling new window input, not retrying a failed
renderer operation with unchanged inputs.
