# Docking drag scenarios on Wayland and X11

The left-edge preview and splitter scenarios failed on Wayland when
their step drivers tried to open an X11 connection. The docking driver
now sends pointer motion, press, and release through the Wayland window
frame loop or XTest on X11. Splitting these phases lets both preview
scenarios inspect the framebuffer with the button still held. The tab
insertion scenario measures the destination again after the floating
panel detaches and the dock reflows.

| Scenario | Steps | Wayland | X11 |
| --- | ---: | --- | --- |
| Left-edge preview and dock | 8 | [Report](wayland/preview.json) | [Report](x11/preview.json) |
| Shared boundary resize and return | 8 | [Report](wayland/resize.json) | [Report](x11/resize.json) |
| Right-side Appearance Browser docking and resize | 13 | [Report](wayland/appearance.json) | [Report](x11/appearance.json) |
| Tab insertion and switching | 10 | [Report](wayland/tabs.json) | [Report](x11/tabs.json) |
| Docking at the right, top, and bottom edges | 11 | [Report](wayland/edges.json) | [Report](x11/edges.json) |

The Wayland captures show the [left-edge target](wayland/left-docking-preview.png)
and the [tab insertion target](wayland/tab-insertion-preview.png) before
release. The scenarios checked the final dock, active document, and
viewport bounds. Both boundary scenarios checked that the original
proportions returned after dragging back.

The build used `fd4816d5e` with the drag driver changes and existing local
filesystem and marking-menu edits. Each scenario ran in a fresh 1440 × 900
session with Vulkan Lavapipe. Wayland used headless Weston, and X11 used
Xvfb and Openbox. [Published input hashes](inputs.sha256)
