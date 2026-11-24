# Ribbon Panel Visibility

Nine native scenarios passed on both benchmark sources, with 105 steps
per source. The direct selector retained its checklist during toggles.
The Tools context menus closed when a panel was selected. Hiding Modify
closed its additional commands, and restoring it kept them closed.

| Scenario | Steps | Pinned | Published |
| --- | ---: | --- | --- |
| Show and hide Parameters from the direct selector | 13 | [Run](pinned/direct/cucumber.json) | [Run](published/direct/cucumber.json) |
| Retain visibility across tabs and Parts | 10 | [Run](pinned/parts/cucumber.json) | [Run](published/parts/cucumber.json) |
| Hide and restore Navigate on View | 12 | [Run](pinned/view/cucumber.json) | [Run](published/view/cucumber.json) |
| Hide Parameters in compact display | 10 | [Run](pinned/compact/cucumber.json) | [Run](published/compact/cucumber.json) |
| Toggle Modify on the left-docked ribbon | 12 | [Run](pinned/side/cucumber.json) | [Run](published/side/cucumber.json) |
| Hide and restore Team Web | 12 | [Run](pinned/team-web/cucumber.json) | [Run](published/team-web/cucumber.json) |
| Hide and restore Tutorials | 12 | [Run](pinned/tutorials/cucumber.json) | [Run](published/tutorials/cucumber.json) |
| Retain a hidden Find panel across tabs | 11 | [Run](pinned/find/cucumber.json) | [Run](published/find/cucumber.json) |
| Toggle Modify with additional commands open | 13 | [Run](pinned/modify/cucumber.json) | [Run](published/modify/cucumber.json) |

The test enables Tutorials through the rendered menu when the scenario
requires it visible. The Part-switch test checks the active document ID
after each tab click.

The application binary used base `dd55e3d8463d6e38a25520a01a96fd84a409dcf4`
with the local step bindings. The native tests ran on X11 with Xvfb, Openbox
and Vulkan Lavapipe. The pinned source was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The
[published snapshot](snapshot.json) verified all 49 files, including both
named Part fixtures. The [published binding inventory](published-bindings.json)
reports 217 unresolved steps across the full feature, down from 280.

The nine pinned declarations gained R0 baseline records.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
