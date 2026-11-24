# Native Ribbon Customization selection and display options

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Transfer an available command after changing the target | [Cucumber](pinned/switch/cucumber.json) |
| Return to a command category after selecting a source row | [Cucumber](pinned/category/cucumber.json) |
| Change command category while an addition is staged | [Cucumber](pinned/draft/cucumber.json) |
| Commit a separator as the only custom panel item | [Cucumber](pinned/separator/cucumber.json) |
| Retain display options when re-adding a staged command | [Cucumber](pinned/options/cucumber.json) |
| Stage the same command with different display options on two targets | [Cucumber](pinned/per-target/cucumber.json) |

Before the bindings, the native [target switch](before/switch/cucumber.json),
[category round trip](before/category/cucumber.json), and
[per-target options](before/per-target/cucumber.json) stopped at undefined
steps. With the steps bound, the
[category probe](probe/category/cucumber.json) reached the transfer
and failed because switching the filter had cleared the source selection.
Both new GUI draft tests failed on the original selection rules.

Switching the target now retains selected source commands that are still
available. A category change keeps a visible selection or selects the
separator if the command is filtered out. Changing the category with no
selection leaves it empty. The
[binding inventory](after-bindings.json) has 534 undefined steps across 126
scenarios, down from [569](../2026-11-24-ribbon-customization-edits-3856/after-bindings.json).

The [selector switching](regression/selectors/cucumber.json) and
[clean reopen](regression/target-reopen/cucumber.json) regressions also passed.
