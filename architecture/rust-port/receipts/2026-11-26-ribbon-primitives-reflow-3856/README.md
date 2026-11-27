# Show Primitives while Modify is expanded

The published Ribbon Panel Visibility scenario opens the unpinned Modify
expansion, shows Primitives from the panel selector, and hides Primitives again.
All 12 steps passed on Wayland and X11. The selector stayed open after both
clicks. Modify and its additional command area each moved 232 pixels to the
right, kept Move Bodies visible, and returned to their original positions.

The adjacent Create reflow scenario passed on both backends. The Primitives
Space toggle also passed on Wayland.

The verification build was based on `24138d2fc` with the new scenario bindings
and existing local GUI filesystem edits. Wayland ran on nested Weston 13 and
X11 ran on Xvfb with Openbox. Both used Vulkan Lavapipe and frame-loop input.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
