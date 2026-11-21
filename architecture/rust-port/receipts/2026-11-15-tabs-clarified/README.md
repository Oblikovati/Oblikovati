# Clarified document tab scenarios

On 2026-11-15, the native application runner used the pinned source snapshot
`8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f`. The document-tabs feature SHA-256
was `2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`.
The test binary was built from the changes committed as `5842f79b1`, on base
`5c42f54f3`. The working tree was uncommitted during the runs.

The Home tab filter passed two scenarios and 17 steps. The window-list filter
passed one scenario and 12 steps. The reports in `home/` and `list/` give the
names and steps. Each run used Linux, Vulkan Lavapipe, Xvfb at 1600 × 1200,
and an isolated Cargo target. The runner injects input into the window frame
loop; these reports do not establish OS-level input delivery.

Both commands used `OBK_BDD_FEATURE=document-tabs`, the respective
`OBK_BDD_SCENARIO` filter, and `cargo test --locked -p oblikovati-head --lib
application_scenarios_use_the_native_window -- --nocapture` under `xvfb-run`.
The filters were `Home tab` and `Activate a document from the tab-strip window
list`.

These are diagnostic results, not verified acceptance receipts. Build
provenance is not bounded; see issue #3978. They do not establish the results
of other document-tabs scenarios.
