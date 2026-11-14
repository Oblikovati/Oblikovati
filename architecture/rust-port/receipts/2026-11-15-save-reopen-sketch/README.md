# Save and reopen a sketch

The pinned scenario passed on 2026-11-15: one scenario and eleven steps passed.
The capture shows the reopened rectangular sketch and its model-browser entry.
This scenario does not require a fully constrained sketch.

Application commit: `81f0ea549bcb2a87011ebdf2214a4a8d9a00f5bf`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run.

```sh
OBK_BDD_ROOT=/tmp/obk-save-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-report-20261115 \
OBK_BDD_FEATURE=document-saving \
OBK_BDD_SCENARIO='Save changes to an existing part and reopen its sketch' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
