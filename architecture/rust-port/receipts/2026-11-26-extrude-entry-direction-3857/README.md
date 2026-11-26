# Part Create Extrude entry and direction

The desktop opened disposable copies of the published fixtures. The checks
cover creating a sketch from the empty Part, all four direction choices,
switching between icon and dropdown controls, cancelling a draft, recovering
from zero distance, changing reference selectors, and opening Parameters from
the distance editor. Direction assertions check the submitted solid bounds
and the visible shaded preview. The Parameters scenario checks the active
document, visible table headers, and unchanged accepted distance after Cancel.
Ten scenarios and 71 steps passed per backend.

The test build used `80cdf0e9c`, the viewport platform guard from
`cb77a3788`, and the [test changes](test-inputs.sha256) in an isolated
checkout. It used Martensite v0.14.2 on Linux. Wayland used the COSMIC
compositor. X11 used Xvfb and Vulkan Lavapipe. Selected coordinates enter
the desktop's viewport event path on Wayland, and the X11-specific controls
retained XTest input. [Published source and fixture hashes](current-inputs.sha256)
matched before and after the test runs. The shared source has no Git revision.

The [reports](reports/) retain Cucumber and JUnit output for the focused runs.
The [native test output](native-x11.txt) also records the existing
named-parameter preview failure. The preview mesh changes, while the
framebuffer comparison fails both before and after the scenario work.

The test target was built with `cargo test --locked -p oblikovati-head --lib --no-run`.
The binary ran `application_scenarios_use_the_native_window` with
`OBK_BDD_AREA=solid-model`, `OBK_BDD_FEATURE=extrusion`, and the scenario
filter in each report filename. The report scope is the listed cases.
