# Native ribbon command list navigation

Issue #3856. These runs use the pinned behavior source
`b576445b98cffd686dcc05aa8c017de2a591c8c8` on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.

| Native scenario | Evidence |
| --- | --- |
| Page between staged commands in a short target list | [Cucumber](pinned/page/cucumber.json) |
| Select visible available commands with the keyboard | [Cucumber](pinned/select-all/cucumber.json) |
| Preserve a staged ribbon edit while viewing keyboard customization | [Cucumber](pinned/staged-keyboard/cucumber.json) |
| Transfer multiple selected commands between customization lists | [Cucumber](pinned/bulk/cucumber.json) |

The initial [paging](before/page/cucumber.json) and
[select-all](before/select-all/cucumber.json) runs stopped at unbound steps.
With the steps in place, the [paging probe](probe/page/cucumber.json)
found that Page Down left the first staged row selected.

Paging now moves by the number of visible rows, stops at the list ends,
and focuses the selected command row. The pinned short-list run checks
selection in both directions, unchanged display options, and cancellation.
The GUI unit test also checks page distances and clamping for longer lists.

The select-all scenario checks the rendered backgrounds of the visible
source rows, including the separator, before and after the key press and
again after the single-row click.
[Target Ctrl+A](regression/ctrl-target/cucumber.json) and
[keyboard transfer](regression/transfer/cucumber.json) passed as regressions.
The [binding inventory](after-bindings.json) has 482 undefined steps across
126 scenarios, down from [505](../2026-11-24-ribbon-customization-keyboard-transfer-3856/after-bindings.json).
