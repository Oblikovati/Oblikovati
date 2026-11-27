# Part marking-menu customization

The published Part profile scenario passed all 17 steps on Wayland and X11.
The editor showed the eight expected command positions, an empty search,
the default style controls, and disabled Apply and Cancel. Close kept
the original Part active and wrote no marking-menu preferences.
[Wayland report](wayland.json) · [Wayland frame](wayland-profile.png) · [X11 report](x11.json)

The initial native run stopped at the missing Part setup binding. With the
new bindings and previous production code, the UI rendered the Part
sub-environment as plain text and displayed the old slots, including raw
command IDs. The new default moves Chamfer to overflow and keeps its
selected-face slot. Revolve is shown disabled pending modeling support.
[Previous layout diagnostic](before.json)

The builds used `9d276d14b` with the changes in this commit and existing
local filesystem and sketch-marking test edits. Wayland used isolated
headless Weston with Pixman. X11 used Xvfb and Openbox. Both runs used
Vulkan Lavapipe. [Published input hashes](current-inputs.sha256)

The targeted GUI and model unit tests, marking preferences tests, and
11 native regressions checked action dispatch, search, Cancel, Apply,
saved settings, and the selected-face menu. [Native checks](native-checks.txt)

The edge-Fillet native test also failed
on the previous executable at its blue-pixel assertion. Both builds
rendered the selected edge as `[45, 212, 191, 255]` with the current theme.
Strict build and runtime provenance remains open under #3978.
