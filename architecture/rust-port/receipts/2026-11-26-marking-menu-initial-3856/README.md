# Inspect the marking-menu customization initial state

The published scenario opens Sketch1 for edit in the named Part. The
native checks verify the 2D Sketch environment, all eight menu positions,
the empty keyword field, the presentation settings, and the disabled
Cancel and Apply controls. The close step compares the active document
and sketch geometry with the state recorded before opening Customize.

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Inspect the marking-menu customization initial state | [17 steps passed](wayland/cucumber.json) | [17 steps passed](x11/cucumber.json) |
| Inspect the keyboard customization initial state | [16 steps passed](wayland/keyboard.json) | [16 steps passed](x11/keyboard.json) |

The Marking Menu run uses the positions configured in Oblikovati for this
fixture, including two empty positions. The keyboard scenario exercises
the shared close check with a different Part that has no sketch in edit.

The build used `449c710c0` with the scenario test changes and existing
local filesystem edits. Wayland used headless Weston. X11 used Xvfb and
Openbox. Both used Vulkan Lavapipe. [Published input hashes](inputs.sha256)
