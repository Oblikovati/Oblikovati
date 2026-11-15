# Body-properties diagnostic run

The pinned body-properties run reported six scenarios and 69 steps passed on
2026-11-15. Review found an empty Given step for unavailable physical
properties. The later dialog assertion checks for N/A, but the setup premise
is not independently established. See
[issue #3954](https://github.com/Oblikovati/Oblikovati/issues/3954). The
appearance-list assertion also checks named entries but does not by itself
prove that the list can scroll.

This is diagnostic evidence, not acceptance for the full feature. The report,
captures and compressed log are retained for review.

Application commit: `fa399e245cb6c1a744225ed5cf679e550e262d69`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree used its own Cargo target; the concurrent agent used a separate target.

```sh
OBK_BDD_ROOT=/tmp/obk-body-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-body-full-report-20261115 \
OBK_BDD_FEATURE=body-properties \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
