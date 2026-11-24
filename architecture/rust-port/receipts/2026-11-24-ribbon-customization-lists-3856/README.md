# Native Ribbon Customization lists and selectors

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Keep staged command order when clicking the target Name header | [Cucumber](pinned/header/cucumber.json) |
| Keep the only staged command in place at ordering boundaries | [Cucumber](pinned/boundary/cucumber.json) |
| Transfer a source command after clicking the empty target list | [Cucumber](pinned/transfer-empty/cucumber.json) |
| Close customization with the command-source list open | [Cucumber](pinned/close/cucumber.json) |
| Press Escape with the command-source list open and a pending edit | [Cucumber](pinned/escape/cucumber.json) |
| Switch between the source and target selectors | [Cucumber](pinned/selector-switch/cucumber.json) |
| Reopen customization after choosing another target without editing | [Cucumber](pinned/target-reopen/cucumber.json) |
| Reopen customization after choosing a command source without editing | [Cucumber](pinned/source-reopen/cucumber.json) |
| Select staged commands with Ctrl+A | [Cucumber](pinned/ctrl-a/cucumber.json) |
| Retain the Marking Menu tab when customization is reopened from Tools | [Cucumber](pinned/marking/cucumber.json) |
| Retain the selected customization tab after reopening | [Cucumber](pinned/keyboard/cucumber.json) |

Before the new bindings, the native runs for the [target header](before/header/cucumber.json),
[empty target click](before/transfer-empty/cucumber.json),
[Close with the selector open](before/close/cucumber.json), and
[source sort](before/sort/cucumber.json) stopped at an undefined step.
The [binding inventory](after-bindings.json) has 625 undefined steps across
126 scenarios, down from [711](../2026-11-24-native-customization-transactions-3856/after-bindings.json).

Two native scenarios reach a command-catalogue gap. The
[ascending order probe](probe/sort/cucumber.json) expects an unlabeled
row and 2-D Drawing before 2D Sketches. The
[descending order probe](probe/sort-transfer/cucumber.json) expects the
four Zoom commands after the separator. The Rust list is backed by the
29 entries in the customization catalogue, which lacks those entries.
Both scenarios remain open in the behavior ledger.
