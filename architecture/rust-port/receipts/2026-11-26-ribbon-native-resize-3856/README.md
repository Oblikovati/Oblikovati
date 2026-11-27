# Native Ribbon Display resize and overflow

Nine published scenarios now pass on both Wayland and X11, with 98 steps
on each backend. They cover native edge resizing and maximizing, tab
overflow, compact panel switching, and Document Settings from both the
full Tools ribbon and its compact Options button.

At the minimum width, the fixture uses 170% text scale to put View in
the overflow menu while Tools remains visible. Checks use the active tab
underline and the command groups visible in the rendered ribbon row.

The full amended Wayland feature passed 36 of 52 cases and 410 steps in
passing cases, up from 27 in the previous run. The combined report
includes a fresh run of the compact-panel tab-switch case after its
Tools assertion was updated. The 16 open cases include tab navigation,
document and Home context transitions, and three unbound scenarios.

The test builds used the application at `960660db8`, the native test
changes in `111780f16`, and existing local GUI filesystem edits. The
published input hashes matched before and after the runs. Wayland used
nested Weston 13 and X11 used Openbox, both on Xvfb with Vulkan Lavapipe.
Strict build and runtime provenance remains open under #3978.

- [Full Wayland report](wayland.json) and [arguments](wayland-arguments.json)
- [Focused X11 report](x11.json) and [arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
