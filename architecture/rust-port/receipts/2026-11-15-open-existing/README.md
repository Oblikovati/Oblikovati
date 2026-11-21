# Activate an already open part

The pinned scenario passed on 2026-11-15: one scenario and nine steps passed.
The capture shows the existing Part tab active and the Drawing tab available.
The scenario checks that the Open command does not create another Part tab.
It does not establish the other document-opening scenarios.

Application commit: `abd5ae18f78bf1e36a6b14bde0c7a5cfaa4b9bee`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run.

```sh
OBK_BDD_ROOT=/tmp/obk-open-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-open-report-20261115 \
OBK_BDD_FEATURE=document-opening \
OBK_BDD_SCENARIO='Activate an already open part through the Open dialog' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
