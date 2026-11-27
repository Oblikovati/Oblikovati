<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Document tabs after off-screen selection fix

On 2026-11-15, the full pinned document-tabs feature passed 38 scenarios and
334 steps on application commit `7b24ae9644a405510b6ae3263c4c93d36aad32f7`.
The source snapshot was `05317bbb2d0809a532b29b7ff6f726a5722b865f`, verified
before and after the run. The feature SHA-256 was
`2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`.

The Xvfb run used the Mesa software Vulkan driver and the native GUI frame-loop
input harness. The JSON and JUnit results, compressed log, source report, and
the corrected overflow capture are retained here. The other full-run captures
remain in `../2026-11-15-document-tabs-full-final/`. The corrected capture was
visually inspected: Presentation is active and all visible part tabs are
neutral. The separate `../2026-11-15-offscreen-tab-selection/` receipt checks
the selected indicator after scrolling back and after a tab click.

This is diagnostic evidence. The harness does not use OS-level input and build
provenance remains open (#3978), so this does not establish verified BDD status.

```sh
OBK_BDD_ROOT=/tmp/obk-tabs-3992-scenarios \
OBK_BDD_REPORTS=/tmp/obk-tabs-3992-full OBK_BDD_FEATURE=document-tabs \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' \
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
