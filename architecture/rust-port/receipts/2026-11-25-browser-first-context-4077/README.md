# Model browser first context click

The GUI frame regression waits for the first idle layout report, then
checks the Sketch row's center after pointer movement, right-button press,
and release. On a fresh Sketch frame and after entering Sketch from Part,
the center stays fixed at 100% and 150% framebuffer scale. The menu's
Visibility action identifies the original row without issuing a selection
action during sketch editing. All 11 browser input tests passed.

The native test opens two sketches and right-clicks the first row while
editing the second. The row keeps its position, and Dimension Visibility
changes on the first sketch while the second remains in edit mode. The
native test passed under Wayland and X11. The separate published
nested-sketch scenario passed all 18 steps on both backends.
[Native and GUI results](test-results.txt) · [Wayland scenario](wayland-cucumber.json) · [X11 scenario](x11-cucumber.json)

The build used `c47a0c887` with these tests and existing local filesystem
and sketch-marking test edits. Wayland used headless Weston with frame-loop
input. X11 used Xvfb and Openbox, with XTest for the standalone right-click.
Both used Vulkan Lavapipe. The source of the historical displacement
reported in issue #4077 remains undetermined. [Published input hashes](inputs.sha256)
