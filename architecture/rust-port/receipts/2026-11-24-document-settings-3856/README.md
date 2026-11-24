# Document Settings from Tools

The full Tools scenario passed all nine steps on both the
[pinned](pinned/cucumber.json) and [published](published/cucumber.json)
sources. The published [width-compacted case](compact/cucumber.json)
passed all eight steps. Each run used its own native X11 window under
Xvfb, Openbox and Vulkan Lavapipe.

The full Tools case verified the six section names, the Standard page,
and Close enabled with Cancel and Apply disabled. Standard reports the
active Part's current assignments. Units retains staged editing. Sketch,
Modeling, Bill of Materials and Default Tolerance are visible and disabled.
The [Units regression run](units.log) passed eight tests.

The [published snapshot](published-snapshot.json) records the named Part
and feature hash. The pinned run used
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The application used revision
`dabd1b4d2` with the local Document Settings change.

The binary built with `cargo test --locked -p oblikovati-head --lib --no-run`
ran `application_scenarios_use_the_native_window --nocapture` with
`OBK_BDD_FEATURE=ribbon-display` and the corresponding
`OBK_BDD_SCENARIO` name. Each run used a fresh `OBK_BDD_REPORTS` directory
and `LP_NUM_THREADS=4`. The pinned case gained an R0 record.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).
