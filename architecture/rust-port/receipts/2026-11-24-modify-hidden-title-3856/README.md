# Modify title strip and selector dismissal

The published cases reproduced the missing title strip on
[restore](before/restore/cucumber.json) and [dismissal](before/dismiss/cucumber.json).
They also reproduced the checklist reopening after a
[second selector click](before/toggle/cucumber.json).

The native implementation retains a compact Modify title with a muted
pin while an unpinned Modify panel is hidden through the direct selector.
Rechecking Modify restores its commands. Clicking the title dismisses
the selector and the strip. The selector button closes its own open
checklist through the same click.

| Native scenario | Steps | Pinned | Published |
| --- | ---: | --- | --- |
| Hide and restore the open unpinned Modify expansion | 11 | [Run](pinned/restore/cucumber.json) | [Run](published/restore/cucumber.json) |
| Dismiss the hidden Modify title strip | 15 | [Run](pinned/dismiss/cucumber.json) | [Run](published/dismiss/cucumber.json) |
| Open and close the direct checklist with its button | 7 | [Run](pinned/toggle/cucumber.json) | [Run](published/toggle/cucumber.json) |
| Hide and restore the panel of a pinned expansion | 16 | [Run](pinned/pinned-expansion/cucumber.json) | [Run](published/pinned-expansion/cucumber.json) |

The published [display-menu handoff](published/menu/cucumber.json) and
[expansion reflow](published/reflow/cucumber.json) also passed all 21 steps.
The [published binding inventory](published-bindings.json) reports 112
unresolved steps, down from 128. Four pinned cases gained R0 records.

The focused runs used app base `e2e41e678` with the local title and
selector changes. They ran with X11, Xvfb, Openbox and Vulkan Lavapipe on
pin `b576445b98cffd686dcc05aa8c017de2a591c8c8` and the verified
[49-file published snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json).

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
