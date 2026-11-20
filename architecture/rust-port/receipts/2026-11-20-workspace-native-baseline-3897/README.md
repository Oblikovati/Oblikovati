# Workspace Layout run with native fixtures

The full run on `782829a16c5756860d04993296015b0ae69bc3b5` passed 15 of
53 cases on Linux under Xvfb and Vulkan Lavapipe. The 52 scenario
declarations include one outline with two cases. Of the 38 stops, 35 were
undefined steps and three were failed assertions. The feature, support
files, and named Part and Drawing fixtures kept the same hashes.

The baseline predates the fixture setup corrections. Three focused runs
then passed the document tab placement, the single Tools tab, and the
default browser width on the saved Part. The separate run on the saved
Drawing passed nine of ten drawing browser cases. The restore case still
stops because the panel-add control is absent after the browser closes.

The source is unversioned and the runtime inputs are unsealed. The reports
are local diagnostics. Strict verification remains open under #3978.

```sh
export OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios
export OBK_BDD_FEATURE=workspace-layout
export OBK_BDD_REPORTS=$(mktemp -d)/reports
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
export VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers"
export XMODIFIERS=@im=none
xvfb-run -a -s '-screen 0 1600x1000x24' \
  cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture --test-threads=1
```
