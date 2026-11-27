# Martensite v0.14 integration

## v0.14.2

The adapter, viewport, and packaging use the
[Martensite v0.14.2 release](https://github.com/Oblikovati/Martensite/releases/tag/v0.14.2).
The annotated tag peels to `d463526642b7bf950ba33c7ab4a4c9bc97caffaf`.
All 20 locked toolkit versions match the tagged manifests, including
`martensite-tkservice` 0.5.1 and `martensite-vulkan` 0.8.1.

The X11 host explicitly requests an X11 display. The new automatic
constructor can return a Wayland display when both display variables are
set. Oblikovati's Xlib calls require an X11 connection. The Linux
head and build-tool check passed for all targets and features. All 113
build-tool unit tests and all three X11 desktop smoke tests passed.

An isolated headless Weston session with Lavapipe presented and captured
a 160 × 120 frame through the new Wayland Vulkan surface with zero
validation errors and no X11 connection. The same check is runnable as
`cargo test --locked -p oblikovati-viewport --test native pinned_martensite_presents_a_native_wayland_frame -- --ignored`
on a Wayland compositor with the Vulkan validation layer.

The [native Wayland host](implementation-wayland.md) wraps an application-owned
surface through this release. The owned Martensite `WlWindow` clears the
drawable before notifying its close listener. Oblikovati keeps its surface
alive through the unsaved-changes prompt and a possible Cancel.

Release preflight verified the dependency metadata and reached the
existing missing resource `packaging/resources/icons/oblikovati.png`.
`cargo audit` reported the same advisory IDs on both lockfiles.
`cargo deny` reported advisory and license failures.

## v0.14.1

The kernel adapter, viewport, and packaging configuration use the published
[Martensite v0.14.1 release](https://github.com/Oblikovati/Martensite/releases/tag/v0.14.1).
The annotated tag peels to `0e72f866844196ae656b23a401c8b81f77b10491`.
Direct and transitive Martensite toolkits resolve to this revision, with each
direct toolkit version taken from its manifest in the tagged tree.

The viewport now supplies typed light and pixel-map handles to the renderer.
Its rendering test backend upgrades weak view handles when delivering a
resize or GUI-layer callback. The geometry adapter compiles against the new
toolkits through its existing operation interfaces.

The Linux workspace check passed with all targets and features. The Windows
x64 MSVC viewport library cross-check passed. Linux X11 with Vulkan Lavapipe
passed the three native desktop smoke tests. All 113 build-tool unit tests
passed.

The adapter and viewport suites recorded 644 passes, five failures, and seven
ignored tests. The five failures also occur with v0.13.1. They cover the
rotated-ellipse pick, curved profile face matching, and three sketch-preview
pixel assertions. The full gate reported the existing 2,183 incomplete R0
scenario records.

Release preflight verified the published SDK and its source before it reached
the existing missing resource `packaging/resources/icons/oblikovati.png`.
The lockfile audit reported the same security advisories as the v0.13.1
lockfile. The workspace dependency policy check reported advisory and license
failures.
