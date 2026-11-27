# Tools panel selector handoffs

The native tests open the Tools selector through its visible control and
verify the destination after a single pointer click. Before the fix,
clicking Customize, Home, or the display arrow dismissed the checklist
without activating the clicked control. Each destination now receives
the click, and the display menu can hand the click back to the selector.

| Native scenario | Steps | Pinned | Published |
| --- | ---: | --- | --- |
| Preserve Tools choices through Home | 15 | [Run](pinned/home/cucumber.json) | [Run](published/home/cucumber.json) |
| Open Customize while the selector is open | 9 | [Run](pinned/customize/cucumber.json) | [Run](published/customize/cucumber.json) |
| Switch between selector and display menu | 10 | [Run](pinned/display-menu/cucumber.json) | [Run](published/display-menu/cucumber.json) |
| Open the Options expansion while the selector is open | 6 | [Run](pinned/options/cucumber.json) | [Run](published/options/cucumber.json) |
| Set Find and Tutorials visibility independently | 13 | [Run](pinned/independent/cucumber.json) | [Run](published/independent/cucumber.json) |

The pinned Home case creates a Part in the session. The published case
opens the named blank native fixture. The tab locator uses the same
whitespace normalization as the drawn text so it can return to
`Blank metric part`. The independent visibility case temporarily reopens
Show Panels to verify its checkmarks, then closes it before the next action.

The [full published run at `3ed3abd45`](full-before-handoff/cucumber.json)
passed 20 of 55 scenarios and 285 of 320 executed steps. Thirty-five cases
stopped at an undefined step. The [updated inventory](published-bindings.json)
reports 166 unresolved bindings, down from 195 in the preceding batch.

The focused runs used application base `1fdc4a5cc` with the local handoff
changes. They ran on X11 with Xvfb, Openbox and Vulkan Lavapipe. They used
pinned source `b576445b98cffd686dcc05aa8c017de2a591c8c8` and the verified
[49-file published snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json).
All five pinned declarations gained R0 evidence.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
