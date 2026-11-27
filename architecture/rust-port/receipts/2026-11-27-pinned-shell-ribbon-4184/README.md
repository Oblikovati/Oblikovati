# Pinned Shell ribbon button sizing

Both scenarios used the clean source at
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and the application code at
`e22f2047ed416e6019e9b0433cd6d3be0d36d624`. X11 used Openbox and Xvfb.
Wayland used headless Weston. Both rendered through Vulkan Lavapipe.

| Scenario | X11 steps | Wayland steps |
| --- | ---: | ---: |
| Change a main-panel command button size | [11](x11-changed.json) | [11](wayland-changed.json) |
| Retain an individual large button when ribbon text is turned off | [12](x11-retained.json) | [12](wayland-retained.json) |

All 23 steps passed on each backend. The checks read the submitted icon
geometry and visible labels and exercised the live size menu.
