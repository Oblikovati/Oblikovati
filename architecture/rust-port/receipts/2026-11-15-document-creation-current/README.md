# Document creation feature diagnostic

The pinned full feature run at application commit
`c88dea3bb442c0f24f0feec5d80e090613930682` completed on 2026-11-15 with
28 passed and 11 failed scenarios. Four failures stopped at undefined steps;
seven stopped at explicit fixture-blocked assertions. The run is diagnostic,
not a native acceptance receipt. It does not establish that later steps in a
failed scenario pass, or that the passing scenarios satisfy every visual premise.

The scenario checkout was clean at
`48987af7b5cbb06515bc6f30385f508382f34064` before and after the run. The
application had the user's unrelated README edit. Xvfb closed when the test
exited. The reports, screenshots, and compressed log record the individual
results. Follow issues #3899, #3951 and #3952 for the remaining work.

```sh
OBK_BDD_ROOT=/tmp/obk-create-next-pinned \
OBK_BDD_REPORTS=/tmp/obk-create-next-report \
OBK_BDD_FEATURE=document-creation \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
