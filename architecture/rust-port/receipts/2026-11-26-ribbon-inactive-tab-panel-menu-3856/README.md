# Show active 3D Model panels from an inactive tab

Right-clicking inactive View or Tools opens Show Panels for the active
3D Model ribbon. Primitives is unchecked, Modify is checked, and the
3D Model command strip remains visible alongside the open menu.

The two outline examples passed all 20 steps on Wayland and X11. The
active-tab docking regression passed all 11 steps on both backends,
including the separate Escape dismissal check.

The verification build was based on `78d1ddab1` with the revised visibility
assertion and existing local GUI filesystem edits. Wayland ran on nested
Weston 13 and X11 ran on Xvfb with Openbox. Both used Vulkan Lavapipe and
frame-loop input.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
