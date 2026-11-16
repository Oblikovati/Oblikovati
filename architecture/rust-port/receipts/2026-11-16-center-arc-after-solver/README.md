<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Center-point arc after solver merge

On 2026-11-16, `arc-center-point-01 Commit the intended result` passed all
seven steps in diagnostic mode at application commit
`49a9d50ea` and pinned scenario snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. The feature SHA-256 was
`35d69dce04958b8bb0b086450fce097d2cc8c99ca367600311c1b6284599469f`.
The snapshot was verified before and after the run. The application worktree
was clean. The report contains the exact step results and source text.

The test used Linux Xvfb at 1920 × 1200 and Vulkan Lavapipe. It used the GUI
frame-loop input harness and did not capture a screenshot. The Xvfb and test
processes exited. This result is diagnostic: it is not a verified BDD receipt,
does not establish OS mouse or keyboard input, and does not bound build
provenance (#3978). It covers only this scenario, not the feature.

```sh
OBK_BDD_ROOT=/tmp/obk-arc-center-scenarios \
OBK_BDD_REPORTS=/tmp/obk-arc-center-report \
OBK_BDD_FEATURE=arc-center-point OBK_BDD_SCENARIO=arc-center-point-01 \
OBK_BDD_DIAGNOSTIC=1 CARGO_TARGET_DIR=/tmp/obk-arc-center-target \
CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
xvfb-run -a -s '-screen 0 1920x1200x24 -noreset' \
  cargo test --offline --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```
