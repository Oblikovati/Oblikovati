# Ribbon available-command keyboard selection

Shift+Down highlights 2D Sketches and 3D Annotations while the Part Tools
target stays empty. Extending the selection through the end of the source
list brings Zoom Window into view. Close dismisses the dialog and
reopening shows the same empty target. The native check passed on Wayland
and X11. [Native results](native-results.txt)

The published selection case passed 11 of 12 steps on each backend. The
final close assertion fails after clicking the disabled Cancel button on
a clean draft. [Source defect](../../scenario-defects/ribbon-shift-down-clean-close.md)
[Wayland report](published-wayland.json) · [X11 report](published-x11.json)

An isolated copy with only the closing step changed to `When the user clicks Close`
passed all 12 steps on Wayland. [Corrected run](corrected-close-wayland.json) · [Corrected source hash](corrected-source.sha256)

The published visible-row selection and multi-command transfer cases
passed all 33 steps on each backend. [Wayland visible selection](visible-wayland.json) · [X11 visible selection](visible-x11.json) · [Wayland transfer](range-transfer-wayland.json) · [X11 transfer](range-transfer-x11.json)

The build used `63241876d` with the new checks and local GUI filesystem
edits. Wayland used headless Weston, and X11 used Xvfb with Openbox. Both
used Vulkan Lavapipe and scripted keyboard events. [Input hashes](inputs.sha256)
