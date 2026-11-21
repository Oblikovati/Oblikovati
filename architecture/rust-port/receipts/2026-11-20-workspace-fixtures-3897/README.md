# Workspace Layout run with the saved Part and Drawing

The full run on `6fa4c41c3d96de8198e216eb8410d156ce23e911` passed 48 of
53 cases and 477 of 482 executed steps. The inventory counts both outline
examples as one declaration, so 47 of 52 declarations passed.

The saved Drawing passed browser restore and log-tab switching. The saved
Parts passed fixture-specific inspection point search, extrusion branch
expansion, and opening the named sketch for edit. The feature, reference
files and fixtures kept the same hashes throughout the run.

Three cases stopped at undefined spacing, View menu, and dock search
steps. Two more used the wrong saved extrusion or sketch name. The
report captures the full run before the later corrections to those cases.

The source is unversioned and the runtime inputs are unsealed. Strict
verification remains open under #3978.

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
