# Pinned marking-menu and model-state coverage

All eight runs used the clean pinned scenarios at
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and Vulkan Lavapipe. X11 ran
under Openbox and Xvfb. Wayland ran under headless Weston. The marking
menu and taper cases used Oblikovati `0e3d32b57558f9eec9834c1bca49dc2a9c5143f0`.
The parameter-tree case used `fb19f5f58a8737a3eb48fc85e4763a9729aa042c`.
Its separate native regression also switched to Copy State and checked that
the highlight moved on both backends.
The menus used `f34ab91573decf3c5f11ef0ca083d7129cbf0efa`. Both native menu
action regressions also passed on each backend.

| Scenario | X11 steps | Wayland steps |
| --- | ---: | ---: |
| Inspect the marking-menu customization initial state | [15](x11-marking-initial.json) | [15](wayland-marking-initial.json) |
| Add and apply a second model-state parameter | [12](x11-state-taper.json) | [12](wayland-state-taper.json) |
| Show active member parameter values in the table tree | [8](x11-state-active.json) | [8](wayland-state-active.json) |
| Inspect parameter column and cell context menus | [10](x11-state-menus.json) | [10](wayland-state-menus.json) |

All 45 steps passed on each backend. The Go tests cited in the
classifications establish the reference behavior by source review.
