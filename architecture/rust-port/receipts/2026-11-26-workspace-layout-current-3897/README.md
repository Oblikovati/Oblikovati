# Workspace Layout on Wayland and X11

All 52 current declarations passed as 53 cases and 523 steps on each
backend. The suite covers a fresh Model browser, shared workspace edges,
ribbon spacing, dock tabs, tree expansion, browser search, context menus,
and the viewport and browser highlights. Both capitalization examples in
the axis-search outline ran separately.

At 1440 logical px the fresh Model browser measured 359 px, about a
quarter of the workspace. Separate Main Window Frame and Docking System runs confirmed
the viewport keeps its reserved space when the window narrows. The side
docks shrink together at minimum size and the active tab, ribbon and
floating feature panel remain usable. The Wayland minimum-size check used
the [private Weston test patch](../2026-11-26-wayland-minimum-frame-4063/weston-13-min-size.patch).

Measurements read the leading menu or indicator mark where the Model and
status labels follow one. The ribbon spacing check counts the visible gaps
at the rendered dividers. Browser checks read the feature chevron, the
active theme's focus outline, the empty search placeholder, and the origin
axis color. These checks distinguish the sketch selection fill from its
adjacent DOF badge. Each combined report includes the focused rerun of
“Filter model browser entries by name”, which checks dismissal using the
search field and clear control.

The head library runner used `OBK_BDD_AREA=application`,
`OBK_BDD_FEATURE=workspace-layout` and `OBK_BDD_ROOT` set to the shared
scenario directory. Wayland used nested Weston 13 and X11 used Openbox.
Both used Xvfb and Vulkan Lavapipe. The feature and recorded supporting
inputs matched their hashes. Issue #3978 tracks strict provenance.

- [Wayland suite](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 suite](x11.json) and [step arguments](x11-arguments.json)
- Search reruns on [Wayland](wayland-search.json) and [X11](x11-search.json)
- Minimum-window checks on [Wayland](minimum-wayland.json) and [X11](minimum-x11.json)
- Side-dock resize checks on [Wayland](dock-resize-wayland.json) and [X11](dock-resize-x11.json)
- [Published input hashes](current-inputs.sha256)
