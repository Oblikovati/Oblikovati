# Quick Access icon and tooltip checks

The published outline passed all 13 examples and 130 steps on each Linux
backend. Native pointer hover displayed the correct tooltip for each icon.
Moving to the viewport dismissed the tooltip and left the same Part active
without opening a command.

The build used `a9fdab737` with local pointer and Extrude test changes and
the existing local GUI filesystem test-module visibility edit. Wayland
ran under nested Weston 13 and X11 under Openbox. Both used Xvfb and
Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after the runs.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
