# Martensite v0.14.1 integration

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
