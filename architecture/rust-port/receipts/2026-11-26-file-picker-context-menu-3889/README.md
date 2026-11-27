# Dismiss a thumbnail context menu

The isolated Wayland run opened the context menu on the published extrusion
thumbnail and failed at the selection count. The failure capture showed the
selected thumbnail near the bottom of the scrolled list. The previous click
offset put the dismissal below the list, so it closed the menu and left the
thumbnail selected.

The binding now clicks the empty gutter within the file list, measured from
the rendered list's inner edge. It checks that the selection count clears and
compares the thumbnail highlight with the gutter on the same row. The final
step still checks the filename and preview.

All ten steps passed on Wayland and X11 against the full shared fixture
directory. This also covers the final case that failed after the earlier
Weston interruption. All 16 interrupted scenarios now have passing isolated
executions.

[Wayland report](wayland.xml) · [X11 report](x11.xml) · [Published inputs](current-inputs.sha256)

The focused runs used base `87aa82ae6de416068b4e7c7a33d799f9bdf8f929`, this
test change, and existing local GUI filesystem and test edits. Wayland used
headless Weston with Pixman. X11 used Xvfb with Openbox. Both used Vulkan
Lavapipe and frame-loop input.
