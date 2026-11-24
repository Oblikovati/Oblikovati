# Direct ribbon panel selector help

The [published case before the fix](before/published/cucumber.json)
found the Show Panels title but failed when it looked for the down-arrow
instruction. The tooltip now explains the down arrow and how checking
or clearing a box changes panel visibility. Clicking the viewport
dismisses it and leaves the Part and its 3D Model ribbon visible.

| Native scenario | Passed steps | Pinned | Published |
| --- | ---: | --- | --- |
| Read and dismisses the direct panel selector help | 10 | [Run](after/pinned/cucumber.json) | [Run](after/published/cucumber.json) |
| Switch between panel visibility and ribbon display menus | 10 | [Run](after/pinned-handoff/cucumber.json) | |

The [native keyboard test](f1.log) opens help with F1 and dismisses it
with Escape. The runs used application base `710ca318f` with the local
help change, X11, Xvfb, Openbox, and Vulkan Lavapipe. The pinned source
was `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The published source was the
[49-file snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json).
The [published binding inventory](published-bindings.json) has 101
unresolved steps, down from 107. The help case gained a pinned R0 record.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
