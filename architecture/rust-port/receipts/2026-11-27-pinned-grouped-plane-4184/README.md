# Pinned Work Features Plane coverage

The clean pinned source was `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The first three cases used the source of `aa515d79e`. The maximize case
used the later setup correction `cf5b8f50e`. All tests rendered through
Vulkan Lavapipe. X11 used Openbox under Xvfb. Wayland used headless Weston
for the first three cases and nested Weston for the maximize case.

| Scenario | X11 | Wayland |
| --- | ---: | ---: |
| Ungroup and regroup a ribbon command family | [9](x11-regroup.json) | [9](wayland-regroup.json) |
| Open and dismisses a grouped command menu | [9](x11-dismiss.json) | [9](wayland-dismiss.json) |
| Launch a choice from a grouped command menu | [9](x11-launch.json) | [9](wayland-launch.json) |
| Reflow the ribbon after changing the application window state | [6](x11-reflow.json) | [6](wayland-reflow.json) |

All 33 steps passed on each backend. The Go tests cited in the
classifications establish the reference behavior by source review.
