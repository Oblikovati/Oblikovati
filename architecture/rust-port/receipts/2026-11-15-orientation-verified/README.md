# Viewport orientation feature run

On 2026-11-15, the pinned orientation feature passed all 20 scenarios and
231 steps at application commit `4fb43878cffd2f9d2a6689600c668b67915cec4b`.
The source checkout was clean before and after the run, and the live feature
hash matched the pinned file afterward. The corner-view capture was inspected.
The native harness injects input into the frame loop; OS-level keyboard and
mouse delivery was not established by this run. Full Go baseline review remains
open in issue #3861.

```sh
OBK_BDD_ROOT=/tmp/obk-3961-snapshot \
OBK_BDD_REPORTS=/tmp/obk-3961-merged-report \
OBK_BDD_FEATURE=viewport-orientation \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
