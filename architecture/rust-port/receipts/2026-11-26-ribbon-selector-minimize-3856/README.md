# Minimize the ribbon while the panel selector is open

A single click closes the selector and minimizes the full ribbon to tabs.
Clicking 3D Model opens its temporary command area. Its selector preserves
Modify checked and Primitives unchecked. A second click on the display
control restores the full ribbon and closes the selector again.

The published scenario passed all 15 steps on Wayland and X11. The panel
context menu minimization regression passed on both, and 12 GUI ribbon
display tests passed. The additional tabs-only scenario reached temporary
command dismissal and failed its selected-tab pixel assertion on both backends.

The verification build was based on `73b232a85` with the display control
fix, new scenario bindings, and existing local GUI filesystem edits.
Wayland ran on nested Weston 13 with frame-loop input. X11 ran on Xvfb
with Openbox and used native clicks for the display control. Both used
Vulkan Lavapipe.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
