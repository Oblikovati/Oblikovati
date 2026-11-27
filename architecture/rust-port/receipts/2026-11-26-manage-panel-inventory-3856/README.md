# Part Manage panel inventory

The published Manage scenario passed all six steps on Wayland and X11.
Show Panels listed Update, Parameters, Manage, Styles and Standards,
Insert, Point Cloud, Layout, Author, iLogic, and Content Center in ribbon
order. All ten were checked, and Manage remained active.

The desktop command test ran Update and opened and closed iProperties
with the same Part active on both backends. Commands without an
implementation appear disabled. The 13-step X11 compaction scenario
verified Insert inline at the wider size, as a compact button after
resizing, and inline again after maximizing.

The full Wayland visibility run passed 52 of 55 cases and 618 steps in
passing cases. The three remaining cases require the side-docked 3D
Model inventory or Shell previews.

The GUI library run ended after reporting 811 results. The remaining
28 tests ran separately with the same binary.

The builds used revision `f3876dc15`, this change, and existing local GUI
filesystem edits. Wayland ran on nested Weston 13 and X11 on Openbox.
Both used Xvfb and Vulkan Lavapipe. The published inputs matched before
and after the runs. Strict build and runtime provenance remains open
under #3978.

- [Wayland inventory](wayland.json) and [X11 inventory](x11.json)
- [X11 Insert compaction](compact-x11.json)
- [Full Wayland visibility report](full-wayland.json)
- [Visibility inputs](inputs.sha256), [compaction inputs](compact-inputs.sha256), and [test results](test-results.txt)
