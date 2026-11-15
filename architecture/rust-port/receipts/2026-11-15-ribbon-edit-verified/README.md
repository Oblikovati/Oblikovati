# Sketch ribbon checks

On 2026-11-15, the native scenario for editing an existing sketch passed all
13 steps at application commit `8c3da8e5925e9f7fcf771f295c177d9ab3350af4`.
The GUI library suite passed 476 tests. The native Return regression passed,
including rectangle sketch finish and undo/redo. These results cover one of the
three ribbon availability scenarios; the outside-edit availability conflict
remains open in [issue #3956](https://github.com/Oblikovati/Oblikovati/issues/3956).

The native harness injects input into the window frame loop. OS-level keyboard
and mouse input was not tested. Tests used Linux, Xvfb, and Vulkan Lavapipe.

```sh
export CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target
export XMODIFIERS=@im=none
export VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json
xvfb-run -a cargo test --locked -p oblikovati-gui --lib -- --test-threads=1
xvfb-run -a cargo test --locked -p oblikovati-head --lib return_finishes_a_rectangle_sketch_and_is_inert_at_part_level -- --test-threads=1
OBK_BDD_ROOT=/tmp/obk-3956-scenarios \
OBK_BDD_REPORTS=/tmp/obk-ribbon-edit-merged-report \
OBK_BDD_FEATURE=ribbon-command-availability \
OBK_BDD_SCENARIO='Inspect Sketch ribbon while editing an existing sketch' \
xvfb-run -a cargo test --locked -p oblikovati-head --lib application_scenarios_use_the_native_window -- --nocapture
```
