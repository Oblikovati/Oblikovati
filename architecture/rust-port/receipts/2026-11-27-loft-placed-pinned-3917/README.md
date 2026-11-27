# Pinned Center Line Loft scenarios

The pinned MCAD-Scenarios source at `b576445b98cffd686dcc05aa8c017de2a591c8c8`
uses implicit starting Parts. The native steps open copies of the same two
packages used by the published fixture variants. The 30 mm guide is used for
creation and the 20 mm guide for intermediate section placement and editing.

| Scenario | Wayland | X11 |
| --- | --- | --- |
| 12 Select the straight guide | [7 steps passed](wayland/loft-12.json) | [7 steps passed](x11/loft-12.json) |
| 13 Place the midpoint section | [6 steps passed](wayland/loft-13.json) | [6 steps passed](x11/loft-13.json) |
| 14 Edit the saved section | [9 steps passed](wayland/loft-14.json) | [9 steps passed](x11/loft-14.json) |
| 15 Cancel an invalid Closed Loop edit | [5 steps passed](wayland/loft-15.json) | [5 steps passed](x11/loft-15.json) |

Both scoped runs used the native application on Linux. The entry for each
display records the [Wayland source revision](wayland/source-revision.json)
and [X11 source revision](x11/source-revision.json).
