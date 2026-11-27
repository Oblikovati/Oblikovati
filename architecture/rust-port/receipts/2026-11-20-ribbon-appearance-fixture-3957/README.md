# Ribbon Appearance fixture verification

The full native window run passed 15 of 18 scenarios and 166 of 169 executed
steps. It tested the source and support files in `inputs.sha256`, which matched
before and after the run. The product and test code matched commit
`2b13407ec` before the receipt was added.

The part-mode scenarios now open a native extruded part and check that the
shaded solid is visible. The Modify expansion starts pinned, as specified in
the fixture notes. It stayed open through Text Off and Normal. The final
capture shows the restored command labels and the visible solid. The runner
used scripted window events and rendered-frame checks under Xvfb and Lavapipe.

Three scenarios still stop at setup for Team Web, Tutorials and User Commands
in Show Panels. The first missing entry was Team Web in all three cases.
The scenario owner has an open request for panel contents and actions in the
feature's `BLOCKERS.md`. This run does not establish release readiness.

The runner used `OBK_BDD_FEATURE=ribbon-appearance` with no scenario filter.

```sh
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```
