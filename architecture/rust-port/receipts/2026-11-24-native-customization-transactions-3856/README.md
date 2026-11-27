# Ribbon customization transactions and secondary selection

The pinned [two-target Apply](before/batch/cucumber.json) and
[reversed transfer](before/reverse/cucumber.json) cases initially stopped
at undefined setup or transfer steps. The native steps now read the
selected target and its visible rows, and exercise the actual selection,
transfer, Apply, Cancel, and reopen controls.

The [secondary selection probe](probe/secondary/cucumber.json) reached
the source list and showed that the clicked command stayed unselected.
The source and target lists now use the same selection handling for
primary and secondary clicks. A native two-row test also verified that a
secondary click changes which staged command gets transferred back.

| Pinned native scenario | Passed steps | Receipt |
| --- | ---: | --- |
| Open ribbon customization from a panel | 6 | [Run](pinned/open/cucumber.json) |
| Stage a command in the selected custom panel | 6 | [Run](pinned/model/cucumber.json) |
| Cancel an uncommitted custom panel edit | 7 | [Run](pinned/model-cancel/cucumber.json) |
| Apply staged commands for two ribbon targets together | 13 | [Run](pinned/batch/cucumber.json) |
| Stage the same command on two ribbon targets | 18 | [Run](pinned/staged/cucumber.json) |
| Apply a customization after reversing a staged command addition | 19 | [Run](pinned/reverse/cucumber.json) |
| Cancel a later edit after applying a custom command | 12 | [Run](pinned/later/cucumber.json) |
| Select an available command without changing the custom panel | 11 | [Run](pinned/available/cucumber.json) |
| Switch customization targets by keyboard with an edit pending | 16 | [Run](pinned/keyboard/cucumber.json) |
| Select an available command with the secondary mouse button | 12 | [Run](pinned/secondary/cucumber.json) |
| Select a staged command with the secondary mouse button | 13 | [Run](pinned/secondary-target/cucumber.json) |

All 133 steps passed and the eleven scenarios gained R0 records. The
pinned source inventory fell from [874 unresolved steps](before-bindings.json)
to [711](after-bindings.json) across 126 scenarios.

The [source-selection replacement case](final/secondary-replace/cucumber.json)
confirmed that the second click changes the selected command, opens no
context menu, and leaves the target empty. It then tried to close with
Cancel, which is disabled with a clean draft as verified in the
[selection-only case](pinned/available/cucumber.json). The replacement
scenario remains open.

The builds used base `a7334f144`, local customization and harness changes,
and existing file-browser edits. The tests ran on Linux X11 with Xvfb,
Openbox and Vulkan Lavapipe. All 44 GUI customization unit tests passed.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
