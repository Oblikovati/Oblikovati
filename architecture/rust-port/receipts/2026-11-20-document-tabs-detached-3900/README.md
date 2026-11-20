# Document tab and detached drawing checks

The detached drawing now keeps its bottom tab and sheet visible when the
main tab strip is hidden. A native X11 drag followed by a click exercised the
detachment path. The separate regression hid the main strip and captured the
remaining drawing sheet and tab.

The full feature run used Linux Xvfb and Vulkan Lavapipe. The drawing drag,
the detached tab context click, and the detached sheet click used X11 events.
Control+Tab and Control+Shift+Tab also used X11. Other steps used generated
native documents and scripted inputs in the desktop window.
The runner reported 39 scenarios and 343 steps passed on `340ff8651`.

The functional notes require an operating-system window for a detached
drawing. The X11 window-tree probe ran during the detachment scenario and
found one titled application window and the 1 × 1 helper window. The drawing
was rendered in the application window. The feature inventory excludes the
three detachment scenarios under #3900 and records 36 supported scenarios.

The source hashes record the live feature and its supporting files. The
run used an unpinned source and unsealed build and runtime inputs, which
prevents strict verification under #3978.

```sh
xvfb-run -a -s '-screen 0 1600x1200x24' cargo test --locked -p oblikovati-head \
  --lib application_scenarios_use_the_native_window -- --nocapture
```
