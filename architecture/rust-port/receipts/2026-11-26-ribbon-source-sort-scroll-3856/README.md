# Reverse the Ribbon Customization command list from a scrolled position

The native regression opens Customize from Part Tools, selects 2D Sketches,
and presses End. Clicking the available-command Name header reverses the
list and brings the separator and descending Zoom commands into view. The
target stays empty and Apply stays disabled until Add transfers the
selected Zoom Window command.

On the baseline `e9a46c1ec`, the reversal left the list scrolled to the
opposite end and displayed the descending tail beginning with Circle.
The fix requests a redraw to show the top of the newly sorted list.
The focused native regression passed on Wayland and X11. Wayland used
headless Weston with scripted window events. X11 used Xvfb and Openbox.
Both used Vulkan Lavapipe. The build also included the existing local GUI
filesystem edits. [Native test results](native-results.txt)

On Wayland the published transfer scenario passed all 14 steps, and the
category-switch and reopen scenario passed all 12 steps. The latter checks
the rendered Zoom and Z Parallel labels in order. Glyph captures omit spaces,
so comparing the concatenated names would misorder them.
[Transfer report](transfer.json) · [Category report](category.json)

The published **Reverse the available command list using its Name header**
case still expects an unlabeled row between the separator and 2-D Drawing.
The functional notes specify the order of the named commands. The unlabeled
row has no identified command. [Published run](published-reverse.json)

[Published input hashes](inputs.sha256) identify the unversioned scenario
and fixtures. Issue #3978 tracks strict build and runtime provenance.
