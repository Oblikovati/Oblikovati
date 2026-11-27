# Native Ribbon Customization import with pending edits

Issue #3856. The runs use behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Earlier run | Passing run |
| --- | --- | --- |
| Import settings while another command is staged on the same target | [Undefined setup](before/same/cucumber.json) | [Cucumber](pinned/same/cucumber.json) |
| Import settings with edits staged on two ribbon targets | [Undefined setup](before/multiple/cucumber.json) | [Cucumber](pinned/multiple/cucumber.json) |

The native chooser opens a scratch XML file with one Part Tools command.
On the same target, the import replaces the staged 3D Annotations row
with the file's 2D Sketches display options. With pending edits on both
targets, Manage remains selected and its draft becomes empty. Tools
shows the imported command and options. Both targets retain pending markers.

The native test `cancel_after_import_restores_applied_ribbon_options`
reopens Customize after canceling and checks that the applied Large and
Text options are still checked.

The [binding inventory](after-bindings.json) has 402 undefined steps across
126 scenarios, down from [422](../2026-11-24-ribbon-customization-draft-files-3856/after-bindings.json).
