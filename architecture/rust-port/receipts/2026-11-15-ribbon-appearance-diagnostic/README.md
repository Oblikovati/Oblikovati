# Ribbon appearance diagnostic run

The pinned ribbon-appearance feature run failed on 2026-11-15. Cucumber reports
seven scenarios passed and 11 failed; 106 steps passed and 11 failed. Nine
scenarios stopped at undefined steps. Two reached assertions that could not
find the expected Team Web group. Steps after each failure did not run. See
[issue #3957](https://github.com/Oblikovati/Oblikovati/issues/3957).

This is diagnostic evidence, not acceptance for the feature. The report,
captures and compressed log are retained for review. The native harness injects
input into the window frame loop; OS-level key and mouse delivery was not tested.

Application commit: `7c5650e8a74a7b8ed5a5aa5ca68a498ca9b4cde0`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree used its own Cargo target; concurrent agents used separate targets.

```sh
OBK_BDD_ROOT=/tmp/obk-ribbon-appearance-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-ribbon-appearance-report-20261115 \
OBK_BDD_FEATURE=ribbon-appearance \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
