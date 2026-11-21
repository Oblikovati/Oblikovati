# Document-creation diagnostic run

The pinned document-creation feature run failed on 2026-11-15. Cucumber reports
29 scenarios passed and ten failed at undefined steps; 306 steps passed and ten
were undefined. The unbound steps include the initial chooser, populated drawing
layout fixtures, and template group and keyboard navigation. Steps after the
first undefined step in each scenario did not run. See
[issue #3899](https://github.com/Oblikovati/Oblikovati/issues/3899) and the
navigation work in [issue #3951](https://github.com/Oblikovati/Oblikovati/issues/3951).

This is diagnostic evidence, not an acceptance receipt. The 29 passing cases
need review and a passing run before they can be counted as current verified
scenarios by the ledger. The report, captures and compressed log are retained
for analysis.

Application commit: `6e62b57608a0ccefcc78c8ecd4efe0423fa0aefa`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. The live feature hash matched the pinned copy after
the run.

```sh
OBK_BDD_ROOT=/tmp/obk-create-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-create-full-report-20261115 \
OBK_BDD_FEATURE=document-creation \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
