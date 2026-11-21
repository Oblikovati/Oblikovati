# Viewport body selection for Clear Appearance

Issue #3962 tracks the remaining Clear Appearance workflow. Viewport body clicks
now pass Ctrl or Shift selection intent through the GUI and owner to the same
ordered body selection used by the model browser. A modified click toggles the
picked body. A modified click on empty space keeps the selection. Unsupported
multi-face and multi-occurrence picks report a selection notice and keep the
current selection.

On Linux, this diagnostic ran with a dedicated Cargo target and Xvfb:

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
CARGO_TARGET_DIR=/tmp/obk-3962-frame-loop-target \
xvfb-run -a -s '-screen 0 1600x1200x24' \
cargo test --locked -p oblikovati-head --lib \
  viewport_two_bodies_clear_with_modifiers -- --nocapture
```

The test used two separate fully constrained sketches and two visible bodies.
It opened Clear, selected the bodies with injected viewport pointer events,
toggled one with Ctrl and Shift, cancelled, applied Clear, and undid the edit.
It passed. These events enter the native frame loop; they are not operating
system mouse and keyboard events or BDD acceptance evidence.

The first Xvfb attempt selected a Radeon driver that could not present to that
window. The next attempt used Lavapipe but exposed duplicate sketch entity IDs
in the test fixture. The passing run used Lavapipe and unique entity IDs.
Exact named-face targets, assembly occurrence clearing, viewport input through
the operating system, and complete BDD verification remain open in #3962.
