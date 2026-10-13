# Native sketch coordinate projection — partial R3

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Reference: Go `app/sketch_tools.go` (`screenToSketch`) at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The native viewport and desktop window now map framebuffer pixels to an authored
sketch plane and map sketch points back to framebuffer pixels. The conversion uses
the current Martensite camera, including pan, zoom, orbit, Fit, Previous View and
surface resize. It does not save a second camera or duplicate the camera matrix
implementation. Coordinates use the same working units as the displayed model.

The forward path calls released `V3d_View::ConvertWithProj`, then the application
adapter calls released `IntAna_IntConicQuad` for the camera line and sketch plane.
The existing sketch frame converts the resulting world point to local coordinates.
The camera line is unbounded, as in Go; this operation does not select B-rep faces,
apply clipping, snap to entities or snap to a grid. An edge-on or ill-conditioned
plane returns no cursor. The positive normal-cosine limit is the Go UI limit of
1e-9, separate from modelling tolerance. The adapter converts Go's inclusive
comparison to the kernel's strict comparison with the next binary64 threshold.
This conversion does not retry an intersection or move geometry.

The inverse path calls released `V3d_View::Convert` for world-to-pixel conversion.
The kernel defines pixel origin, inverted Y and rounding. The application checks
that the projected pixel is finite and representable before the kernel's integer
conversion; it reserves one integer of rounding headroom at each limit. Invalid
coordinates, unusable directions and unrepresentable intersection results are
explicit errors. Negative/offscreen pixels remain supported when representable.

A pending native resize cannot combine the new surface dimensions with the old
camera aspect: projection refuses until the normal event poll observes the resize.
The queries do not change scene/camera generations or invalidate a completed frame.
Closed views refuse both operations.

Tests cover XY/XZ/YZ and translated tilted planes, positive and negative line
parameters, multiple working scales, edge-on/near-edge-on directions and invalid
inputs. Independent orthographic coordinates check the native pixel convention;
pixel-to-plane-to-pixel tests allow at most one pixel of integer rounding. Native
captures check that a projected interior point lies on the rendered triangle.
The same checks run after navigation, resize and Previous View. A recorded device
checks refusal and recovery across an unobserved resize.

A native desktop test maps two pixel locations through the actual document sketch
plane, feeds those points into the constrained slot recipe and commits through the
existing owner. It verifies the owned centers and that undo restores the original
viewport pixels. This establishes the coordinate path to the existing commit path;
it does not claim that a mouse tool or preview widget is already connected.

Tool activation, click/drag collection, inference, live recipe drawing and editable
field widgets remain open. General profile-to-solid construction remains a critical
kernel blocker. Martensite stays pinned to v0.2.5 and the API to v0.158.0, with no local
source override. Windows/macOS native runtime verification remains open.

The full local `make gate` passed for all 13 crates, all targets and features:
formatting, Clippy, build, tests, documentation tests, coverage, Markdown/link
checks, SPDX, workflow lint and duplication. Native tests ran on a
1920×1080×24 Xvfb display with software Vulkan and validation layer 1.4.321.
Every crate exceeded 80% line coverage. The new kernel adapter reached 93.94%,
viewport projection 96.88%, and desktop forwarding 100%. Duplication remained
0.77% (545/70,654 scanned lines), with no new detected clone.

The cross-platform and comment review checked signed pixel bounds, finite values,
framebuffer versus logical coordinates, native view lifetime and authored working
units. No geometry is moved to make projection succeed, and there is no app-owned
intersection fallback. The resize test uses the existing named recorded device.
This change is local only, with no push or release.
