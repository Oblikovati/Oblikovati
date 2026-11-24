# Empty Part viewport after reopen

The pinned Save and reopen case reproduced the full gate's viewport
assertion failure [with](before-openbox/cucumber.json) and
[without](before-gate/cucumber.json) Openbox. The reopened Part was active
and the browser showed no sketches or features. The previous test rectangle
extended into the docked Appearance browser.

The check now samples the measured document viewport, inset from its
docking edges. It still requires an empty feature and sketch list and
uncovered pixels with the viewport background gradient.

All twelve steps passed [with](after-openbox/cucumber.json) and
[without](after-gate/cucumber.json) Openbox on X11 with Xvfb and Vulkan
Lavapipe. The focused runs used base `3ed3abd45` with the viewport
assertion change and benchmark source
`b576445b98cffd686dcc05aa8c017de2a591c8c8`.

See [#3888](https://github.com/Oblikovati/Oblikovati/issues/3888).
