# Create a selected part template with Enter

The pinned scenario passed on 2026-11-15: one scenario and eight steps passed.
The capture shows the new Part tab, origin entries, and end-of-part marker.
It does not establish the behavior of the other document-creation scenarios.

Application commit: `3b97dd00f5a7ea3bc8ec6a6b85536ee6824225b0`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run. The live feature hash also matched the pin.

```sh
OBK_BDD_ROOT=/tmp/obk-create-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-create-pinned-report-20261115 \
OBK_BDD_FEATURE=document-creation \
OBK_BDD_SCENARIO='Create a selected part template with Enter' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
