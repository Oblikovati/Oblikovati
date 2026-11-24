# Native Ribbon Customization edits and import

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Reorder and remove staged commands | [Cucumber](pinned/reorder/cucumber.json) |
| Apply and remove a custom ribbon panel | [Cucumber](pinned/remove-panel/cucumber.json) |
| Switch targets while an edit is pending | [Cucumber](pinned/switch/cucumber.json) |
| Commit a custom command with OK | [Cucumber](pinned/ok/cucumber.json) |
| Stage and remove repeated separators | [Cucumber](pinned/separators/cucumber.json) |
| Insert a separator after a selected custom panel row | [Cucumber](pinned/insert/cucumber.json) |
| Retain a staged ribbon edit while viewing marking menu customization | [Cucumber](pinned/marking/cucumber.json) |
| Return to customization after selecting malformed XML | [Cucumber](pinned/malformed/cucumber.json) |

The earlier runs stopped at undefined steps for
[reordering](before/reorder/cucumber.json), [applying a panel](before/remove-panel/cucumber.json),
[repeated separators](before/separators/cucumber.json), and
[malformed XML](before/malformed/cucumber.json). The
[binding inventory](after-bindings.json) has 569 undefined steps across
126 scenarios, down from [625](../2026-11-24-ribbon-customization-lists-3856/after-bindings.json).

With the native import step bound, malformed XML
[left the file picker open](probe/malformed/cucumber.json). The picker now
closes and Customize displays the import error. A native test also checks
that 2D Sketches remains staged and the target stays marked as pending.
A [missing file](regression/missing/cucumber.json) still warns in the picker.
A [valid export](regression/valid/cucumber.json) still stages its settings
for review before Apply.
