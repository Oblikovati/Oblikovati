# Solid-face selection with native Linux input

All four published solid-face scenarios passed 24 steps on each backend.
The checks cover face highlighting, the pick cue, the contextual toolbar,
the standard and radial menus, and clearing the selection from empty
viewport space. The runner now sends these clicks through the active
Linux desktop. The prior Wayland run failed at the first face click
because the runner looked for an X11 application window.

The standalone test passed on Wayland and X11. The three deferred menu
entries kept the selection intact, and Properties opened the face
property dialog. The Ribbon Customization scenario also passed on
Wayland with its selected face preserved through opening and closing
Customize.

Four cases in each full report stopped at 3D sketch setup. They require
presentation and picking of saved spatial geometry. Two also require
3D sketch editing (#4092).

The build used revision `9223d33a8`, the test change, and existing
local GUI filesystem edits. Wayland used nested Weston 13, and X11 used
Openbox. Both ran under Xvfb with Vulkan Lavapipe. The published input
hashes matched before and after verification. Strict build and runtime
provenance remains open under #3978.

- [Wayland before the input change](wayland-before.json)
- [Wayland report](wayland.json) and [X11 report](x11.json)
- [Wayland pointer trace](wayland-pointer.txt)
- [Ribbon Customization scenario](ribbon-wayland.json)
- [Input hashes](inputs.sha256) and [test results](test-results.txt)
