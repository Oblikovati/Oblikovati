# Double-click the application bar to maximize and restore

The published scenario passed all seven steps on Wayland at 1920 × 1080
and at 3840 × 2160 with 2× scaling. The gesture sent one maximize request
and one restore request per run. The application showed Restore while
maximized and Maximize after restoring. The test matched the original
desktop position and size against the compositor's scene graph.
[Wayland report](wayland.json) · [4K Wayland report](wayland-4k.json)

The published X11 scenario also passed all seven steps. Weston ran its
desktop shell on a private compositor for the Wayland runs. X11 used
Xvfb, Openbox and tint2. All runs used Vulkan Lavapipe. The focused native
Wayland checks also passed for the double-click gesture and the Maximize
and Restore buttons.
[X11 report](x11.json) · [Geometry and protocol trace](test-results.txt)

The baseline `c52279f89` attempted to connect to X11 when the Wayland run
reached the first double-click. The passing build used backend-aware test
input, the Weston scene-graph probe, and existing local GUI filesystem
edits. The published source hashes matched before and after.
[Baseline report](baseline.json) · [Input hashes](inputs.sha256)

The [BDD workflow](../../bdd-workflow.md) describes how to supply the
Wayland compositor probe for this scenario.
