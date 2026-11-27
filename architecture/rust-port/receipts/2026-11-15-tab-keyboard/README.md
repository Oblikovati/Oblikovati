# Document-tab keyboard switching

Four pinned scenarios passed on 2026-11-15, with 29 passing steps. They check
forward and backward switching, Home and wrap behavior. The native test harness
injects physical-key events into the window frame loop; OS-level key delivery
was not tested. The final captures were reviewed. These cases do not establish
acceptance of the full document-tabs feature.

Application commit: `df87baea493c33badfe81ec4ebab3dfb9a250891`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after each run. Go baseline details remain open for these rows.

```sh
OBK_BDD_ROOT=/tmp/obk-tab-3950 OBK_BDD_FEATURE=document-tabs \
OBK_BDD_REPORTS=/tmp/obk-tab-merged-keyboard OBK_BDD_SCENARIO=keyboard \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
OBK_BDD_ROOT=/tmp/obk-tab-3950 OBK_BDD_FEATURE=document-tabs \
OBK_BDD_REPORTS=/tmp/obk-tab-merged-backward \
OBK_BDD_SCENARIO='Switch backward and forward between neighboring part tabs' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
