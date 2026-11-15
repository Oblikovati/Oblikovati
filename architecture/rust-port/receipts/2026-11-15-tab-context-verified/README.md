# Document tab context checks

On 2026-11-15, two pinned scenarios passed on application commit
`e9f07c43bc2ec9f77c51cea515e16d8df68ed899`: the tab context menu (six steps)
and properties for an inactive saved part (11 steps). Each run has its own
report and capture. The harness injects input into the frame loop and does not
prove OS-level keyboard or mouse delivery. The full document-tabs feature and
the detailed Go baseline remain open in issues #3900 and #3861.

```sh
export OBK_BDD_ROOT=/tmp/obk-tab-context-3900
export OBK_BDD_FEATURE=document-tabs
export CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target
export XMODIFIERS=@im=none
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
OBK_BDD_REPORTS=/tmp/obk-tab-context-merged-3900 \
OBK_BDD_SCENARIO="Inspect a document tab's context menu" \
xvfb-run -a cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture
OBK_BDD_REPORTS=/tmp/obk-tab-properties-merged-3900 \
OBK_BDD_SCENARIO='Inspect properties of an inactive document from its tab' \
xvfb-run -a cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture
```
