# Native Model Browser disclosure controls

Four published scenarios passed all 39 steps on Wayland and X11. They
cover expanding and collapsing Origin, an extrusion, all browser branches,
and the solid bodies folder. The feature scenario also verifies that
clicking the disclosure arrow preserves the existing feature selection.
The test locates the arrow to the left of the rendered feature icon.

The build used `a3739fdab635e25aeb0bcc19637aa03c21adb0b8` with the existing
local GUI filesystem test-module visibility edit. Wayland used nested
Weston 13, X11 used Openbox, and both used Xvfb and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source before
and after the runs. Strict build and runtime verification remains open
under #3978.

- Wayland [report](wayland.json) and [resolved arguments](wayland-arguments.json)
- X11 [report](x11.json) and [resolved arguments](x11-arguments.json)
