# Save and close cancellation

Four pinned scenarios passed on 2026-11-15, with 30 passing steps. The run
checks cancelling a modified part close, cancelling the first Save As, and
cancelling changes to the preview checkbox and preview source. The preview
steps verify the intermediate changes before cancellation (#3948).

Application commit: `8497c121fdbc58e1e22be233108c637fd6971c0b`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. Each final capture was reviewed.

```sh
OBK_BDD_ROOT=/tmp/obk-preview-cancel-scenarios \
OBK_BDD_REPORTS=/tmp/obk-preview-cancel-merged-report \
OBK_BDD_FEATURE=document-saving OBK_BDD_SCENARIO='Cancel ' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
