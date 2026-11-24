# Native Ribbon Customization entry points

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Open ribbon customization from the Tools tab in its first captured state | [Cucumber](pinned/tools/cucumber.json) |
| Open customization from the View ribbon context menu | [Cucumber](pinned/view/cucumber.json) |
| Cancel a staged command on the View ribbon target | [Cucumber](pinned/view-cancel/cucumber.json) |
| Open Ribbon customization from the Sketch panel title | [Cucumber](pinned/sketch/cucumber.json) |
| Open and close customization while one face is selected | [Cucumber](pinned/face/cucumber.json) |

Earlier native runs stopped at undefined steps for the
[first Tools opening](before/tools/cucumber.json),
[View title menu](before/view/cucumber.json),
[Sketch title menu](before/sketch/cucumber.json), and
[single-face selection](before/face/cucumber.json).

The steps open the dialog through the rendered ribbon. They check the
active target, clean controls, and the state left behind on closing.
The face scenario checks the active solid Part and the one-face indicator.
The shared clean-dialog assertion also passed the
[Document Settings regression](regression/settings/cucumber.json).

The [binding inventory](after-bindings.json) has 447 undefined steps across
126 scenarios, down from [482](../2026-11-24-ribbon-customization-list-navigation-3856/after-bindings.json).
