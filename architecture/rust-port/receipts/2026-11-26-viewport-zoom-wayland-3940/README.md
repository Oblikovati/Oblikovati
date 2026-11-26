# Native mouse-wheel zoom on Wayland

All seven published Viewport Zoom scenarios and 74 steps passed on both
backends. Three wheel notches zoomed out, and three in the opposite
direction zoomed back in. The same Part and camera orientation stayed
active throughout.

The Wayland wheel moved through the private Weston desktop shell. Its
pointer trace records the discrete vertical scroll events in both
directions. The X11 run used XTest for the wheel. The other Zoom actions
used input through the native window's frame loop.

The build used application revision `2b4c55ca9`, the test changes, and
existing local GUI filesystem edits. Wayland used nested Weston 13 with
a 2048 × 1200 Xvfb host, and X11 used Xvfb and Openbox. Both used Vulkan
Lavapipe. Input hashes matched before and after verification, and focused
Clippy passed. Strict build and runtime provenance remains open under #3978.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Wayland wheel trace](wayland-wheel.txt) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
