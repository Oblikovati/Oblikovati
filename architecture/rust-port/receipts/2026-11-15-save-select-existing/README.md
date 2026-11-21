# Select an existing part in Save As

The pinned scenario passed on 2026-11-15, with eight passing steps. The native
assertions check row highlighting, the selected filename and the selection
count. The reviewed final capture shows the Part after cancelling the dialog;
it does not show the intermediate selected row. The file is a temporary test
fixture and is not opened or overwritten by this scenario.

Application commit: `651ea05d134cbd90e78d2fa79850265d205857d1`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. Go baseline details for this row remain open.

```sh
OBK_BDD_ROOT=/tmp/obk-save-select-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-select-report-20261115 \
OBK_BDD_FEATURE=document-saving \
OBK_BDD_SCENARIO='Select an existing part in the first-save dialog' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
