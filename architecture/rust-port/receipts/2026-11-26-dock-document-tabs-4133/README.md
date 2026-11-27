# Retain a tabbed dock while switching documents

The published scenario opens the blank PartOne and the Part containing
a solid block. It opens Log from the Model Browser dock menu, selects
Log for PartOne and Model for the solid Part, then switches through both
document tabs again. It checks that the same left dock contains both
panels, the selected panel follows the active document, and the viewport
alternates between the shaded block and the blank workspace.

| Linux backend | Baseline | With per-document tab selection |
| --- | --- | --- |
| Wayland | [11 steps passed, then Log remained inactive](wayland/baseline.json) | [13 steps passed](wayland/cucumber.json) |
| X11 | [11 steps passed, then Log remained inactive](x11/baseline.json) | [13 steps passed](x11/cucumber.json) |

The floating Appearance Browser scenario and the native dock-menu test
also passed on both backends. They check that a floating panel retains
its position and that opening and closing Log leaves Model available.
[Regression results](test-results.txt)

The baseline used `d8079404e` with the new scenario bindings. The final
build added the tab-selection change. Both builds included existing local
filesystem edits. Wayland used headless Weston. X11 used Xvfb and Openbox.
Both used Vulkan Lavapipe. [Published input hashes](inputs.sha256)
