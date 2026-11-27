# Pinned compact Options panel dismissal

Both scenarios used the clean source at
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and the application code at
`ee7201d823a34e73826dac614b0787cc616b13d4`. X11 used Openbox and Xvfb.
Wayland used headless Weston. Both rendered through Vulkan Lavapipe.

| Scenario | X11 steps | Wayland steps |
| --- | ---: | ---: |
| Press Escape with a compact Options panel open | [11](x11-escape.json) | [11](wayland-escape.json) |
| Dismiss a compact panel using the selected tab | [7](x11-selected.json) | [7](wayland-selected.json) |

The three neighboring compact Options cases also passed on both backends.
The two matching published scenarios passed all 19 steps per backend using
their named Part fixtures.
