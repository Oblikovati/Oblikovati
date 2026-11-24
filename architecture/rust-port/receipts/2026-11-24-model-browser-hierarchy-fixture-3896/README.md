# Model Browser hierarchy on the named Part

Four published cases now keep `workspace-layout-native.opd` active.
The test steps read the extrusion and sketch names from the browser.
Expanding the feature branch leaves sketch visibility intact. The separate
sketch context-menu setup still prepares a hidden profile.

| Native scenario | Pinned | Published |
| --- | --- | --- |
| Collapse and expands the origin branch | [10 steps passed](pinned/origin/cucumber.json) | [11 steps passed](published/origin/cucumber.json) |
| Select a solid feature in the browser | [9 steps passed](pinned/select/cucumber.json) | [10 steps passed](published/select/cucumber.json) |
| Collapse and expands a feature branch | [8 steps passed](pinned/branch/cucumber.json) | [9 steps passed](published/branch/cucumber.json) |
| Inspect the extrusion row context menu | [10 steps passed](pinned/menu/cucumber.json) | [11 steps passed](published/menu/cucumber.json) |

The [published nested-sketch](published/nested-sketch/cucumber.json) and
[pinned Workspace Layout](pinned/workspace-sketch/cucumber.json) cases also
passed. The X11 feature-outline test
[reproduced the missed deselection](x11-before.log) and
[passed after picking an unobstructed viewport point](x11-after.log).

The native runs used `92c40a26e` with the local test changes and existing
file-browser edits on Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe.
The pinned source was `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The [published manifest](../2026-11-24-model-browser-named-part-3896/snapshot.json)
records the named fixture. The four pinned scenarios gained R0 records.
See [#3896](https://github.com/Oblikovati/Oblikovati/issues/3896).
