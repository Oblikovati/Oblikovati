# Retain a floating Appearance Browser while switching documents

The native scenario opens the published blank Part and the Part containing
a solid block. It opens Appearance Browser from Quick Access and records
its floating position. It switches through the document tabs, checks the
Model Browser against the active document, and checks the rendered viewport
as the solid disappears and returns. The Appearance Browser stays open in
the same position after both switches.

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Retain a floating appearance browser while switching documents | [10 steps passed](wayland/cucumber.json) | [10 steps passed](x11/cucumber.json) |

The existing layout-retention scenario also passed all eight steps on
X11 after the two-document setup gained the new step wording.
[Shared setup regression](x11/tabbed-layout.json)

The baseline stopped at the first step because the two-document setup had
no matching binding. The verified build used `d19288b3d`, the new scenario
bindings, and existing local filesystem edits. Wayland used headless
Weston. X11 used Xvfb and Openbox. Both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256)
