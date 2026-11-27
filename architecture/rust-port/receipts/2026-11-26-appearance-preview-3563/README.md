# Appearance Editor preview scenes and lighting

Preview Options opens Scene, Environment, and Render Settings submenus.
Sphere and Cube render the current appearance draft. The Environment
submenu applies Grid Light, Plaza, Snowfield, or Warm Light, and
Render Settings shows Quick selected. Preview choices belong to the
open window and leave the document appearance clean. The remaining
sample shapes are listed with their controls disabled.

Both new scenarios passed all 15 steps on Wayland and X11. The native
Warm Light test verified the changed framebuffer colors, the selected
menu item, and the clean appearance. Checker texture and Transparency
each passed their nine-step regression on both desktops.

The build used base `b128e6748`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb with
Vulkan Lavapipe. The runs used a copy of the [published scenario and
fixture inputs](published-inputs.sha256). The shared Functional notes
were edited during the run to update their summary of other captured
appearance types. The preview requirements, feature, and fixture were
unchanged. Strict provenance remains open under #3978.

- Scene on [Wayland](scene-wayland.json) and [X11](scene-x11.json), with the [Wayland cube screenshot](cube-wayland.png)
- Environment and Render Settings on [Wayland](lighting-wayland.json) and [X11](lighting-x11.json)
- Native Warm Light check on [Wayland](warm-wayland.txt) and [X11](warm-x11.txt)
- Checker on [Wayland](checker-wayland.json) and [X11](checker-x11.json)
- Transparency on [Wayland](transparency-wayland.json) and [X11](transparency-x11.json)
