# Document-opening diagnostic run

The pinned document-opening feature run failed on 2026-11-15. Cucumber reports
26 scenarios passed and one failed; 261 steps passed and one was undefined.
The undefined step is the saved assembly with visible components in the
express-loading scenario. Later steps in that scenario did not run. See
[issue #3949](https://github.com/Oblikovati/Oblikovati/issues/3949).

This is diagnostic evidence, not an acceptance receipt. The 26 passing cases
need review and a passing run before they can be counted as current verified
scenarios by the ledger. The report, captures and compressed log are retained
for analysis.

Application commit: `835e118ccc6c223ed22b7acad2b6b19a63ba3374`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. The live feature hash matched the pinned copy after
the run.

```sh
OBK_BDD_ROOT=/tmp/obk-open-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-open-full-report-20261115 \
OBK_BDD_FEATURE=document-opening \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
