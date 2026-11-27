# Ribbon tab context and panel placement

The Tools context-menu scenario reproduced an inactive 3D Model tab
becoming active on a right-click. The ribbon now keeps the active tab
selected through the right-button press and release. Show Panels retains
the inventory and check states of the active Tools ribbon.

| Native scenario | Steps | Pinned | Published |
| --- | ---: | --- | --- |
| List active Tools panels from an inactive tab menu | 9 | [Run](pinned/inactive-tools/cucumber.json) | [Run](published/inactive-tools/cucumber.json) |
| Hide and restore Sketch Pattern | 13 | [Run](pinned/sketch/cucumber.json) | [Run](published/sketch/cucumber.json) |
| Reopen the sketch with Pattern still hidden | 10 | [Run](pinned/reopen/cucumber.json) | [Run](published/reopen/cucumber.json) |
| Retain independent visibility for side-docked tabs | 13 | [Run](pinned/side-tabs/cucumber.json) | [Run](published/side-tabs/cucumber.json) |
| Reflow the open Modify expansion after hiding Create | 11 | [Run](pinned/reflow/cucumber.json) | [Run](published/reflow/cucumber.json) |

The Sketch case reopens the same Profile row through the model browser and
compares its recipe before finishing and after reopening. The reflow case
compares the positions of both the Modify caption and its expansion before
hiding Create and after restoring it.

The [pinned Ribbon Appearance regression](appearance/cucumber.json)
passed all 16 steps. It toggles panel titles from an inactive tab menu
and checks the Options title when its expansion arrow is drawn alongside it.

The focused runs used base `8deaf75f2` with the local tab and binding
changes. They ran on X11 with Xvfb, Openbox and Vulkan Lavapipe against
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and the verified
[49-file published snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json).
The [binding inventory](published-bindings.json) reports 128 unresolved
steps, down from 166. Five pinned scenarios gained R0 evidence.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
