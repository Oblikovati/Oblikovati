# Ribbon Panel Expansion

Five published scenarios passed all 61 steps on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe. Each run selected one scenario by name.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Open the Tools Options additional command area | 11 | [Cucumber](options-open/cucumber.json), [capture](options-open/capture.png) |
| Cancel the layout reset from the Options expansion | 9 | [Cucumber](options-reset/cucumber.json), [capture](options-reset/capture.png) |
| Minimize the ribbon while an expanded panel is pinned | 12 | [Cucumber](options-cycle/cucumber.json), [capture](options-cycle/capture.png) |
| Minimize to tabs after pinning the Modify expansion | 19 | [Cucumber](modify-minimize/cucumber.json), [capture](modify-minimize/capture.png) |
| Close the Constrain expansion when finishing sketch edit | 10 | [Cucumber](sketch-finish/cucumber.json), [capture](sketch-finish/capture.png) |

The three Options cases cover the named blank Part, outside-click dismissal,
the reset confirmation's No button, and clearing the pin during a display
cycle. A [separate X11 input test](options-x11.log) clicked the reset
command and the No button. It captured the [open Options panel](tools-options-expanded.png)
and [reset confirmation](tools-options-reset-confirmation.png).
The Modify case checks that the pin clears when minimizing and that
the temporarily opened panel dismisses with the ribbon. The Constrain case
checks that finishing the saved Sketch1 retains the same sketch recipe and
returns to the solid Part.

The [local snapshot](snapshot.json) records the published feature and
fixture inputs under `dfbe3564d8e78386834e955d833ad0f6d5c06ea5`. The
feature SHA-256 was
`3941cd9f652313f9209c5c9fe64f881414f6939a36591d725798b8e73ba39345`.
The 54 recorded hashes matched both the snapshot and published files after
the runs. Each case directory includes its native log, JUnit, and source
revision report.

The application used `741e50fadf271ef05ad36312b29ddabc29dd2c52` plus
working-tree changes for the Options expansion, saved sketch fixtures, and
ribbon assertions. The API was pinned at `907cd4c28713b47501b077b8d902dbb6a008aada`
and Martensite at `bef25f47548431c8f128f08f66a1f719bcee4982`.

The runs used the binary built with
`cargo test --locked -p oblikovati-head --lib --no-run` and this invocation
under the private X11 display. Each run set `OBK_BDD_SCENARIO` to a name
in the table and used a fresh `OBK_BDD_REPORTS` directory.

```sh
export XMODIFIERS=@im=none
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
export LP_NUM_THREADS=4
export OBK_BDD_ROOT=/tmp/oblikovati-ribbon-relocated.RcZVDl/snapshot/checkout
export OBK_BDD_FEATURE=ribbon-panel-expansion
target/debug/deps/oblikovati_head-422152f5f0046b19 \
  application_scenarios_use_the_native_window --nocapture
```

The [binding inventory](step-bindings.json) found 80 unresolved steps
across nine published cases, down from 102 across 12. The
[pinned application inventory](pinned-application-bindings.json.gz) found
1,316 unresolved steps, down from 1,335 before the Options work. Shell preview work
still depends on [Martensite #554](https://github.com/Oblikovati/Martensite/issues/554).
The published-source runs are diagnostic evidence.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

## Compact Options panel

Four focused Ribbon Display scenarios also passed all 41 steps on the same
X11 and Lavapipe setup. The compact Options panel displays its reset action
with the main commands, stays open after Escape and another group-button
click, and closes on the lower title or a click outside the panel.

| Scenario | Steps | Report |
| --- | ---: | --- |
| View additional Options commands from a compact panel button | 9 | [Cucumber](compact-display/display-compact-options/cucumber.json) |
| Dismiss a compact panel using its lower title | 7 | [Cucumber](compact-display/display-options-lower-title/cucumber.json) |
| Press Escape with a compact Options panel open | 11 | [Cucumber](compact-display/display-options-escape/cucumber.json) |
| Repeat the active compact panel button | 14 | [Cucumber](compact-display/display-options-repeat/cucumber.json) |

The [Ribbon Display snapshot](compact-display/snapshot.json) used revision
`9241d437e7cad18412942240d25f9e58c0fde911` and feature SHA-256
`72e8e750b3cfef027f9acf3ed894074eb597c34d366deaec03714bd3dbe08b62`.
All 35 recorded hashes matched both the snapshot and published files
after the runs. The test binary used application revision
`b575cc4264eb6eeae2ec50d86e6902f3e9c87b9c` plus the full and compact
Options changes. The command above used the Ribbon Display snapshot as
`OBK_BDD_ROOT`, `ribbon-display` as `OBK_BDD_FEATURE`, and each case
name as `OBK_BDD_SCENARIO`.

The [Ribbon Display binding inventory](compact-display/step-bindings.json)
found 37 unresolved steps across eight scenarios, down from 41 across ten.
