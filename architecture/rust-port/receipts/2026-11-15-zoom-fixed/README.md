# Viewport zoom after selected-face fix

On 2026-11-15, the pinned zoom feature passed seven scenarios and 74 steps at
application commit `34ca0f952b33e313a3bd2cb7d1f1c10a16b2292e`. The two Zoom
Selected scenarios now check the boundary of the clicked face in the document
area. Three native regressions cover preselection, selection after command
start, and perspective; the selected-face capture was inspected. The source
checkout stayed clean and the live feature hash matched the pinned file.

The native harness injects input into the frame loop. This run does not prove
OS-level keyboard or mouse delivery. The full Go baseline remains open in
issue #3861.

```sh
export CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target
export XMODIFIERS=@im=none
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  viewport_zoom::face_selection_regression -- --test-threads=1
OBK_BDD_ROOT=/tmp/obk-quick-access-3955-source \
OBK_BDD_REPORTS=/tmp/obk-zoom-final-merged-report \
OBK_BDD_FEATURE=viewport-zoom xvfb-run -a cargo test --locked \
  -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture
```
