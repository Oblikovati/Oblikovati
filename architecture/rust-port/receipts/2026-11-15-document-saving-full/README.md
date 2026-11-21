# Document-saving feature run

The pinned document-saving feature passed on 2026-11-15: 23 scenarios and 218
steps passed. The run covers first save, reopen, cancellation, preview controls,
file and folder selection, prompt choices, and save-on-close. The final captures
were reviewed. Some scenarios finish after closing the dialog, so their final
captures do not show intermediate states; the native assertions check those
states. The solid preview scenario creates a solid through the native UI.

Application commit: `9461065e35459ae8c8b18b9f574ef8dc1689b267`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. The live feature hash matched the pinned copy after
the run. Missing Go baseline details remain open in the classifications.

```sh
OBK_BDD_ROOT=/tmp/obk-save-full-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-save-full-report-20261115 \
OBK_BDD_FEATURE=document-saving \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
