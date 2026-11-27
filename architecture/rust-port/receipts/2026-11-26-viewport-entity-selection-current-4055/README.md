# Current native viewport entity selection

All four solid-face scenarios passed on each Linux backend, with 24 steps
in passing cases. They cover the face and boundary highlight, the pick cue,
the contextual toolbar, both context menus, and clearing the selection.
The four spatial-sketch cases stopped at setup. They still require desktop
presentation and picking of saved 3D sketch geometry. Two also require
editing an active 3D sketch (#4092).

The build used `96b2086bbc58e3849efdb3a842b2d92093706069` and the existing
local GUI filesystem test-module visibility edit. Wayland used nested
Weston 13, X11 used Openbox, and both used Xvfb and Vulkan Lavapipe. The
[feature, reference, and fixture hashes](current-inputs.sha256) matched
the shared source before and after both runs. Strict build and runtime
verification remains open under #3978.

- Wayland [report](wayland.json) and [resolved arguments](wayland-arguments.json)
- X11 [report](x11.json) and [resolved arguments](x11-arguments.json)
