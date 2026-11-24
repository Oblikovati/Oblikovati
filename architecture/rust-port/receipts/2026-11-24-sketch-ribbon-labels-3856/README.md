# Sketch ribbon labels and appearance

The full Ribbon Display run at `c5c250779` passed 46 of 50
scenarios. In the command-label case, the Normal precondition left the
Sketch Create appearance submenu open after Escape, and the next right
click dismissed the open menu. The focused published run reproduced the
failure at the third step.

The precondition now verifies that Normal is checked and closes the
menu with a viewport click. The case then opens the menu, hides command
labels, restores them, and checks that the active sketch stays unchanged.

| Native run | Passed steps |
| --- | ---: |
| [Hide and restore labels, pinned](pinned/cucumber.json) | 10 |
| [Hide and restore labels, published](published/cucumber.json) | 10 |
| [Inspect appearance choices with Escape, published](escape/cucumber.json) | 11 |
| [Change command size modes, pinned](sizes/cucumber.json) | 18 |

The focused runs used application base `00bedf8a87e11e9083052b2242047d9367f63bec`
with the local precondition change. They ran on X11 with Xvfb, Openbox,
Vulkan Lavapipe and `LP_NUM_THREADS=4`. The pinned source was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The published runs used
the [35-file Ribbon Display snapshot](../2026-11-24-ribbon-display-full-3856/snapshot.json).
The pinned command-label case gained an R0 record.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
