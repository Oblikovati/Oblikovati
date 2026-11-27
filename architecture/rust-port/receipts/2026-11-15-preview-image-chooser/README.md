# Cancel the preview image chooser

The pinned scenario passed on 2026-11-15, with 14 passing steps. The native
assertions check the chooser order and the selected import option after the
chooser closes. The reviewed final capture shows the Part still active after
Save As was cancelled. It does not show the intermediate chooser state.

Application commit: `07ba93fca5fe108614f442de4a43e68bf917044d`.
The existing local `README.md` edit was present and did not affect the build.
The scenario checkout was clean at `48987af7b5cbb06515bc6f30385f508382f34064`
before and after the run.

```sh
OBK_BDD_ROOT=/tmp/obk-image-cancel-pinned-20261115 \
OBK_BDD_REPORTS=/tmp/obk-image-cancel-report-20261115 \
OBK_BDD_FEATURE=document-saving \
OBK_BDD_SCENARIO='Browse for a preview image and cancel the image chooser' \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
