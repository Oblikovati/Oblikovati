# Save a modified part while closing

The pinned scenario passed on 2026-11-15, with 12 passing steps. It saves a
new modified Part while closing, activates another document, and reopens the
saved Part. The reviewed capture shows the reopened Part and its empty sketch.

Application commit: `c5b37477a370c24c91d7bf71ea57a78f68b983d0`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run.

```sh
OBK_BDD_ROOT=/tmp/obk-save-close-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-close-report-20261115 \
OBK_BDD_FEATURE=document-saving \
OBK_BDD_SCENARIO='Save a new modified part while closing it' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
