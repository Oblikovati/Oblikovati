# Rectangle visibility with UI occlusions

On 2026-11-15, the native runner used scenario snapshot
`8ec8d2c1e569fbc5c8cd2ec85e9b9380c62c147f` and the test changes committed as
`6a076b0d7` on base `7075016f1`. The test changes were uncommitted when run.
The feature SHA-256 values were
`2738bff62d9968e32a59d426e9a3e762938981a2bdb7e074f77bdf59c5d4928f`
for document tabs and
`ee58ca1d01af2e1db700be89fa237cee19f7d42c477fbe6c8f5f62d07683cbd2`
for document saving.

- Switch between open parts passed all 13 steps.
- Save changes to an existing part and reopen its sketch passed all 11 steps.
- Seven rectangle visibility unit tests passed, including an occluded negative
  observation and a genuinely absent rectangle.
- The first-part native journey passed, including hiding and restoring the
  rectangle through a transaction and testing both observed states.

The scenarios ran with the selected `OBK_BDD_FEATURE` and `OBK_BDD_SCENARIO`
filters through `cargo test --locked -p oblikovati-head --lib
application_scenarios_use_the_native_window -- --nocapture`. The native journey
used the filter `native_new_part_button_creates_and_selects_an_unsaved_document`.
The runs used a dedicated Cargo target, Linux, Vulkan Lavapipe, and Xvfb. They
injected input into the window frame loop, rather than OS-level keyboard and
mouse delivery. The scenario results are diagnostic, not verified receipts;
runner build provenance remains open in #3978.
