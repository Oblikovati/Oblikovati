# Project dialog feature run

The pinned project-dialog feature passed on 2026-11-15: eight scenarios and
101 steps passed. The run covers project information, inactive projects,
new-project wizard navigation, folder selection, and project tree expansion.
The final captures were reviewed. The native harness injects input into the
window frame loop; OS-level key and mouse delivery was not tested.

Application commit: `f104b9fd6c109b47e3a3507dac3cb2cc09efa7f1`.
Only documentation and receipt changes were committed during the run. The
existing local `README.md` edit did not affect the build. The scenario checkout
was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and after the
run, and the live feature hash matched the pin afterward. Go baseline details
remain open where not already recorded.

An earlier shared-target run also passed, but is diagnostic only because other
worktrees were building concurrently. This run used an isolated Cargo target.

```sh
OBK_BDD_ROOT=/tmp/obk-project-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-project-isolated-report-20261115 \
OBK_BDD_FEATURE=project-dialog CARGO_TARGET_DIR=/tmp/obk-project-isolated-target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
