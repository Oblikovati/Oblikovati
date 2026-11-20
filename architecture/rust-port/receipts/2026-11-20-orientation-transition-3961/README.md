# Viewport Orientation desktop acceptance

The current scenarios use the saved rectangular Part
`fixtures/cad/docking-visible-solid-current.opd`. The setup opens that file
through the desktop Open dialog. It hides the Profile overlay and clears
its selection before measuring the visible solid. The file stays unchanged.

Application commit `7550ae9f2561ef0c0c5e894a5fc49a3c36e0218b`.

The ViewCube animation follows the enabled preference. The focused native
run passed all seven steps with an X11 click under Openbox. It recorded an
intermediate frame with the solid and orientation control visible. Both
reached Top together while the same document and scene generation stayed
active. The viewport unit suite passed 182 tests, with one ignored. It
included a turn between opposite views.

All 23 scenarios and 253 steps passed under Xvfb with Vulkan Lavapipe.
All-target Clippy passed
for the desktop, GUI, and viewport crates. A Windows GNU cross-compile
checked the desktop crate. The live window checks ran on Linux.

The input hashes cover the feature, supporting files, and the native Part.
The live scenario source and the build and runtime inputs were unsealed.
Strict verification remains blocked under #3978.

```sh
export OBK_BDD_ROOT=/mnt/vininas-projects/MCAD-Scenarios
export OBK_BDD_FEATURE=viewport-orientation
export OBK_BDD_REPORTS=$(mktemp -d)/reports
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
export VK_LAYER_PATH="$PWD/.tools/vulkan-validation-1.4.321/layers"
export XMODIFIERS=@im=none
tools/with-xvfb.sh cargo test --locked -p oblikovati-head \
  --lib application_scenarios_use_the_native_window -- --nocapture
```
