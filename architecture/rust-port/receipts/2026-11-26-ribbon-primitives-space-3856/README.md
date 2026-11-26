# Toggle the Primitives panel with Space

Primitives starts hidden in the 3D Model ribbon and appears between Sketch
and Create in the selector. Enabling the panel exposes the disabled
Box, Cylinder, Sphere and Torus controls.
[Extrusion primitive reference](https://help.autodesk.com/cloudhelp/2022/ENU/Inventor-Help/files/GUID-B0CFA8E7-D3F5-48A9-BE6F-C8DF668301BD.htm) · [Revolve primitive reference](https://help.autodesk.com/cloudhelp/2024/ENU/Inventor-Help/files/GUID-401BABD2-8D5B-4300-BA8D-70037445044F.htm)

The published case opens the selector, presses Down twice to reach
Primitives, and presses Space twice. The first press adds the panel to
the ribbon. The second hides it again. The selector remains open with
the same focused row until Escape dismisses it.

The verification build was based on `6385c664b` and included the existing
local GUI filesystem edits. Wayland ran on nested Weston 13 and X11 ran
on Xvfb with Openbox. Both used Vulkan Lavapipe and frame-loop input.
[Input hashes](inputs.sha256)

All 14 steps passed on Wayland and X11. Three native selector regressions,
12 GUI ribbon-display tests, and workspace Clippy also passed.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Visible panel](primitives-visible.png) · [Test results](test-results.txt)
