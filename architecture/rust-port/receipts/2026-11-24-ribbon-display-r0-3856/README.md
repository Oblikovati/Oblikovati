# Ribbon display and Options baselines

Three published Ribbon Display scenarios passed on Linux X11 under Xvfb,
Openbox and Vulkan Lavapipe. The published feature SHA-256 was
`72e8e750b3cfef027f9acf3ed894074eb597c34d366deaec03714bd3dbe08b62`.
The [published snapshot](published-snapshot.json) records both native Part
fixtures and the feature source.

| Published scenario | Steps passed | Report |
| --- | ---: | --- |
| Access a ribbon tab in tabs-only mode | 15 | [Cucumber](published-tabs/cucumber.json) |
| Stop wheel navigation at the visible ends of the part ribbon | 10 | [Cucumber](published-wheel/cucumber.json) |
| Dismiss the ribbon appearance submenu with Escape | 7 | [Cucumber](published-escape/cucumber.json) |

All reports used application base `0a1a620a171288ae924a15fb601b6e3cd70ae624`
with local scenario bindings. The API pin was
`907cd4c28713b47501b077b8d902dbb6a008aada` and the Martensite pin was
`bef25f47548431c8f128f08f66a1f719bcee4982`.

## Pinned differences

The restored snapshot was `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The pinned Ribbon Display SHA-256 was
`82542b170cbae71ede004fcb11be64a14bedbb10fdaf9b40df7b2eec462f636e`.

Seven pinned cases passed all 68 steps. Their R0 records retain the
benchmark expectations and evidence for the current pin.

| Pinned scenario | Steps passed | Report |
| --- | ---: | --- |
| View additional Options commands from a compact panel button | 9 | [Cucumber](pinned-compact/cucumber.json) |
| Dismiss the ribbon appearance submenu with Escape | 7 | [Cucumber](pinned-escape/cucumber.json) |
| Dismiss a compact panel using its lower title | 7 | [Cucumber](pinned-lower-title/cucumber.json) |
| Repeat the active compact panel button | 14 | [Cucumber](pinned-repeat/cucumber.json) |
| Open the Tools Options additional command area | 10 | [Cucumber](pinned-full-options/cucumber.json) |
| Minimize the ribbon while an expanded panel is pinned | 12 | [Cucumber](pinned-options-cycle/cucumber.json) |
| Cancel the layout reset from the Options expansion | 9 | [Cucumber](pinned-options-reset/cucumber.json) |

The pinned Ribbon Panel Expansion SHA-256 was
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`.
Strict acceptance remains open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).

- The pinned tabs-only case passed 13 steps, then failed its final full-ribbon
  assertion after two display activations. The published sequence restores
  the full ribbon with one activation after choosing Minimize to Panel Buttons.
  [Pinned native log](pinned-tabs/native.log).
- The pinned wheel case passed five steps, then reached an undefined step for
  selecting Fusion. The published case targets the last visible command tab,
  which is Annotate in the Rust Part ribbon. [Pinned native log](pinned-wheel/native.log).

Each case ran on a private display opened with `tools/with-xvfb.sh`, with
`VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json` and
`LP_NUM_THREADS=4`. The binary built by
`cargo test --locked -p oblikovati-head --lib --no-run` ran
`application_scenarios_use_the_native_window --nocapture`.
`OBK_BDD_ROOT` selected the restored pin or published snapshot, and
`OBK_BDD_FEATURE` and `OBK_BDD_SCENARIO` selected one case. Each case
directory contains its source revision, arguments, native log, Cucumber
and JUnit outputs.

See the [Go ribbon baseline](../2026-11-14-go-ribbon-baseline/README.md)
for panel width and fit evidence and [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856)
for the Rust ribbon benchmark.
