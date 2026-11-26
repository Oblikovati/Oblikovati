# Ribbon keyboard navigation with the current Part fixture

All 18 published scenarios and 220 steps passed on Wayland and X11. The
runs covered ribbon tab and command key tips, two-letter command entry,
unmatched keys, panel and display menus, temporary and side-docked ribbon
groups, model-browser navigation, view rotation, and solid-face selection.

The native test build used application revision `bcd51cda7` and included
the existing local GUI filesystem edits. Wayland ran on nested Weston 13
and X11 ran on Xvfb with Openbox. Both used Vulkan Lavapipe and frame-loop
input. The current Part fixture and feature support files matched the recorded
hashes at the end of the run. Strict build and runtime provenance remains
open under #3978.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Wayland scenario arguments](wayland-arguments.json) · [X11 scenario arguments](x11-arguments.json) · [Input hashes](inputs.sha256)
