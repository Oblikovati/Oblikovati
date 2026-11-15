# Viewport orientation diagnostic run

The pinned viewport-orientation feature run failed on 2026-11-15. Cucumber
reports two scenarios passed and 18 failed at undefined steps; 71 steps passed
and 18 were undefined. Unbound steps include face, corner, arrow and roll
navigation as well as options and projection interactions. Steps after the first
undefined step in each scenario did not run. See
[issue #3961](https://github.com/Oblikovati/Oblikovati/issues/3961).

This is diagnostic evidence, not acceptance for the feature. The report,
captures and compressed log are retained for review. The native harness injects
input into the window frame loop; OS-level key and mouse delivery was not tested.

Application commit: `b51592db0b6d3c3e19c5bc0dff22dbacdc04ea5d`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree used its own Cargo target; concurrent agents used separate targets.

```sh
OBK_BDD_ROOT=/tmp/obk-orientation-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-orientation-full-report-20261115 \
OBK_BDD_FEATURE=viewport-orientation \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
