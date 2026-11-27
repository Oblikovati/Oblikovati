# Sketch Text presets

Pinned `text-24` passed all eight steps through the native Format Text window.
The X11 test opened the preset list, verified No Preset and Last Used, opened
inline naming with `ab`, and pressed Escape. Format Text and the preview stayed
open. The Preset settings check verified the three disabled current-preset
actions and the checked Recently Used sort and No Preset default.

- [Pinned Cucumber report](pinned-text-24/cucumber.json)
- [Preset settings capture](settings.png)

A separate native check saved `ab` as a session preset, canceled the draft,
and applied the preset to a fresh placement. Saving the preset made no sketch
geometry. After accepting the fresh placement, Last Used restored `ab` in the
next draft. The preset library belongs to the current editing session.

The runs used Linux X11 under Xvfb and Vulkan Lavapipe. The pinned check used
`OBK_BDD_FEATURE=text` and `OBK_BDD_SCENARIO=text-24` with the verified scenario
snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`.

The full live feature run passed eight of 38 cases. The other 30 stopped at
undefined steps. The scenario specification retained its recorded hash.
[Cucumber report](cucumber.json)

All five native Sketch Text regressions and the two preset unit tests passed.
Strict acceptance still needs bounded build and runtime inputs under #3978.
