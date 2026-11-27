# Ribbon Display run and focused reruns

The full published feature passed 45 of 50 cases and 481 of 486 executed
steps. Five cases stopped at undefined steps. The [Cucumber report](full/cucumber.json)
covers the complete run, including the width-compacted Insert case,
the Sketch Create appearance case, the full Document Settings case,
and two Shell preview cases.

The full run used application revision
`8c8b311e40613985c5f71381fafdf5311c3b210b` and the clean
[published snapshot](snapshot.json) at
`9241d437e7cad18412942240d25f9e58c0fde911`.
All 35 snapshot hashes matched the published source and its named fixtures
after the run. The Ribbon Display feature SHA-256 was
`72e8e750b3cfef027f9acf3ed894074eb597c34d366deaec03714bd3dbe08b62`.

## Focused reruns

The focused binary used base `74258ad5a4a5b52e37915d6d44c09fc4871ebf69`
with the Sketch Create Escape change and BDD bindings in this batch.
Each case passed against both the published source and the restored
pin `b576445b98cffd686dcc05aa8c017de2a591c8c8`.

| Scenario | Steps per run | Pinned | Published |
| --- | ---: | --- | --- |
| Open a panel compacted by a narrow application window | 12 | [Cucumber](pinned-manage/cucumber.json) | [Cucumber](published-manage/cucumber.json) |
| Inspect ribbon appearance choices without changing the display mode | 11 | [Cucumber](pinned-appearance/cucumber.json) | [Cucumber](published-appearance/cucumber.json) |
| Dismiss the ribbon appearance submenu with Escape | 7 | [Cucumber](pinned-modify-escape/cucumber.json) | [Cucumber](published-modify-escape/cucumber.json) |

The Insert fixture uses 175% icon scale because the default-scale panel
fits at the X11 minimum width of 640 pixels. The test uses the native window
manager to narrow and maximize the window. The outside click uses viewport
bounds, so the narrowed case dismisses from inside the application.
Only the Sketch Create title's appearance submenu retains Escape. The
Part Modify title still dismisses both menu layers, and a click outside the
Create title menu leaves the active sketch and its geometry unchanged.
The two newly passing pinned cases have R0 records.

All runs used X11, Xvfb, Openbox and Vulkan Lavapipe with
`VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json` and
`LP_NUM_THREADS=4`. The binary built with
`cargo test --locked -p oblikovati-head --lib --no-run` ran
`application_scenarios_use_the_native_window --nocapture`.
`OBK_BDD_ROOT` selected a source, `OBK_BDD_FEATURE` was `ribbon-display`,
and focused runs set `OBK_BDD_SCENARIO` to the table's scenario name.
Each run used a fresh `OBK_BDD_REPORTS` directory.

The full result tracks its tested revision. The focused evidence covers
its three table rows. See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856)
and [Martensite #554](https://github.com/Oblikovati/Martensite/issues/554).
