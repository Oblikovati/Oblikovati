# Jump to Z Parallel in Ribbon Customization

The All Commands catalog includes Z Parallel with a stable settings identifier.
A user can arrange it in a custom panel. Its visible button is disabled and
its tooltip identifies the command as unavailable because the 3D Sketch
constraint is still deferred.

The published scenario opens Customize on the Part Tools target, selects
2D Sketches, and presses Z. The native check looks for the newly visible
Z Parallel row and compares its highlight with the nearby Zoom All row. The
target stays empty and Apply stays disabled. The settings round-trip test
includes a staged Z Parallel entry.

The first run with the new entry selected Zoom. The sort removed the space
in Z Parallel and put it after the Zoom commands. Preserving spaces puts
Z Parallel first among the Z commands in ascending order.

The published scenario passed all ten steps on Wayland and X11. The native
source-list paging regression passed on both backends and all 58 ribbon
customization unit tests passed.

[Wayland report](wayland.xml) · [X11 report](x11.xml) · [Published inputs](current-inputs.sha256)

The build used base `a8f03ae2d3cfb00c9ecd3be391b89d4f8d2b6bd0`, this change,
and existing local GUI filesystem and test edits. The Wayland run used
headless Weston with Pixman and the X11 run used Xvfb with Openbox. Both
used Vulkan Lavapipe and frame-loop input.
