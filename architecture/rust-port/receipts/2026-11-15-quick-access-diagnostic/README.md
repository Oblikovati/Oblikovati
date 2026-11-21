# Quick access toolbar diagnostic run

The pinned quick-access-toolbar feature run failed on 2026-11-15. Cucumber
reports 11 scenarios passed and one failed; 122 steps passed and one failed.
The customization command-choice scenario could not find Refine Appearance;
later steps in that scenario did not run. See
[issue #3955](https://github.com/Oblikovati/Oblikovati/issues/3955).

This is diagnostic evidence, not acceptance for the feature. The report,
captures and compressed log are retained for review. The native harness injects
input into the window frame loop; OS-level key and mouse delivery was not tested.

Application commit: `b94c65c7ce26e22b7ee71d2211d3e070a9d62b36`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree used its own Cargo target; the concurrent agent used a separate target.

```sh
OBK_BDD_ROOT=/tmp/obk-qat-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-qat-full-report-20261115 \
OBK_BDD_FEATURE=quick-access-toolbar \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
