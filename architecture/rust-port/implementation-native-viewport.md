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
polled. A size change during redraw is an error and requires a new frame.
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
