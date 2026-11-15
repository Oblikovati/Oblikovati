# Body-properties feature run

The pinned body-properties feature passed on 2026-11-15: six scenarios and 69
steps passed. The native checks establish unavailable values before opening,
verify exact calculated rows, exercise appearance-list scrolling, and check
appearance apply and cancellation. The solid fixture is a 40 × 30 × 20 mm box
with density 1 g/cm³; its expected volume, area, mass and center are calculated
independently from those dimensions. Final captures were reviewed; they show
the part after the dialog closes rather than intermediate dialog states.

Application commit: `f3fb65521ca173242cf5202051d0e2246eec2283`.
The existing local `README.md` edit did not affect the build. The scenario
checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064` before and
after the run, and the live feature hash matched the pin afterward. The main
worktree used its own Cargo target; concurrent agents used separate targets.
The native harness injects events into the window frame loop; OS-level input
was not tested.

```sh
OBK_BDD_ROOT=/tmp/obk-3954-snapshot \
OBK_BDD_REPORTS=/tmp/obk-body-values-merged-report OBK_BDD_FEATURE=body-properties \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

The focused `calculated_physical_rows_reject_missing_non_numeric_and_partial_values`
unit regression also passed on the same commit. This receipt does not establish
per-face kernel validation or complete the overall solid-model feature work.
