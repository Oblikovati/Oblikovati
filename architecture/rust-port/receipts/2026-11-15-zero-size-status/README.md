# Zero-size preview status

Issue #3975. Application commit: `a96424d63`. The earlier live screenshot in
`../2026-11-15-rectangle-zero-corrected/zero_locked.png` shows the internal
geometry error after entering zero and pressing Tab. `red.log` reproduces that
status in the native scenario before the fix. That red run used the live,
unversioned scenario source; its feature hash matched the pinned snapshot.
A second negative control, `timing-red.log`, used the pinned source and shows
that the error appears immediately after typing `0`, before Tab.

The Go reference obtains an in-progress rectangle preview from
`app/sketch_preview.go` through `RectangleTool.PendingRecipe`. This state does
not need a placement error. The Rust fix omits the invalid typed preview and
keeps the placement prompt. It does not suppress other preview errors.

The pinned run used clean scenario commit
`8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f` and passed one scenario with eight
steps. `owner.log` covers both zero dimensions and a later valid rectangle.
Only the selected scenario was run.

```sh
OBK_BDD_ROOT=/tmp/obk-3975-source OBK_BDD_REPORTS=/tmp/obk-3975-pinned \
OBK_BDD_FEATURE=rectangle-two-point OBK_BDD_SCENARIO=rectangle-two-point-10 \
CARGO_TARGET_DIR=/tmp/obk-3975-target \
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' \
  cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

A separate XTest mouse and keyboard run created a metric Part and XY sketch,
started Rectangle, clicked a first corner, typed `0`, pressed Tab, typed `4`,
pressed Return, and pressed Escape. The MCP Bridge captured the window and
read the sketch entities and constraints. `first_corner.png`, `typed_zero.png`
and `zero_locked.png` show the status at each stage. `rejected.png` and
`final.png` show the next prompt and Ready. The entity and constraint reports
match before and after. The window and Xvfb were closed after the run.
