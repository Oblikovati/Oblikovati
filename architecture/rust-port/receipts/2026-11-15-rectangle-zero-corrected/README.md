# Corrected two-point rectangle scenario

Application test commit: `e66db6b3d`. Scenario source: clean commit
`8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f` in MCAD-Scenarios.
The scenario feature SHA-256 is
`c5b51038c64283e9ec2a81e892ea8f912a06264a80eee7e42d65932f3146aa59`.
The live feature had the same hash after the test.

The native frame-loop BDD test passed one scenario and eight steps. It used a
real metric Part and XY sketch. It verifies the locked first field, focused
second field, unchanged sketch recipe and undo/redo history. The report and
final screenshot are included. Other scenarios were filtered out.

```sh
OBK_BDD_ROOT=/tmp/obk-rect-corrected-source \
OBK_BDD_REPORTS=/tmp/obk-rect-corrected-final \
OBK_BDD_FEATURE=rectangle-two-point \
OBK_BDD_SCENARIO=rectangle-two-point-10 \
CARGO_TARGET_DIR=/tmp/obk-rect-corrected-target \
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' \
  cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

A separate live GUI test used XTest mouse and keyboard input and the MCP
Bridge for screenshots and read-only sketch inspection. At 1900 × 1100 it
created a metric Part, started an XY sketch, chose Rectangle, clicked a clear
point, entered `0`, pressed Tab, entered `4`, pressed Return, then pressed
Escape. `zero_locked.png` shows the locked 0.000 mm field and focus on the
second field. `rejected.png` shows Select first corner without a dialog.
`final.png` shows Ready without a rectangle. The entity and constraint
reports are identical before and after. The GUI and Xvfb were closed after the
run. The XTest driver and MCP probe are included for inspection; the scripts
contain temporary paths specific to this run.
