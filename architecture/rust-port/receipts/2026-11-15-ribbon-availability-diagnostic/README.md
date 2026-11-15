# Sketch ribbon availability diagnostic run

The pinned ribbon-command-availability feature run failed on 2026-11-15. All
three scenarios stopped at the undefined inactive-sketch setup step. Three
setup steps passed and three were undefined. The command-availability and
editing assertions did not run. See
[issue #3956](https://github.com/Oblikovati/Oblikovati/issues/3956).

This is diagnostic evidence, not acceptance for the feature. The report,
captures and compressed log are retained for review. The native harness injects
input into the window frame loop; OS-level key and mouse delivery was not tested.

Application commit: `c69ba6a105643a1828fb416b9ff191104a439af9`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree used its own Cargo target; concurrent agents used separate targets.

```sh
OBK_BDD_ROOT=/tmp/obk-ribbon-availability-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-ribbon-availability-report-20261115 \
OBK_BDD_FEATURE=ribbon-command-availability \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
