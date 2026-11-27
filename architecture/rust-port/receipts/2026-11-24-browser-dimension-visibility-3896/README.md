# Sketch and dimension visibility from the browser

All 17 steps passed in the [pinned native run](cucumber.json). It checked
that sketch and dimension visibility can be toggled independently.
Clicking empty viewport space kept annotations on the visible sketch.
Hiding the sketch then cleared the annotations and left the solid visible.

The older full gate stopped when the annotations remained after the
final hide. This run used the viewport-point helper from the
[browser deselection fix](../2026-11-24-browser-viewport-deselection-3896/README.md).

The build included `e7994401f` and the existing local file-browser edits.
The test ran under X11, Xvfb, Openbox, and Vulkan Lavapipe against
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The pinned scenario gained
an R0 baseline record. See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
