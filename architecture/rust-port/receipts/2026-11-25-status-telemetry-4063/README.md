# Status telemetry and sketch constraint color

The 24 px status bar now uses 10 px JetBrains Mono for pointer coordinates,
DOF, constraint messages, units and zoom. The 6 px status dot starts at
the 10 px left inset. Fully Constrained uses the published success color
for the active Studio palette.
[Before](before-dark.png) · [Dark](after-dark.png) · [Light](light.png)
· [High Contrast](high-contrast.png)

The native sketch test failed on the old neutral color. It now verifies
the success color in all three palettes and the separate DOF cell. Drawing
a rectangle changes the message to the neutral dimension count while the
tool prompt stays visible. [Dimensions needed](dimensions-needed.png)

The native Part test still measured the 24 px bar and verified pointer
coordinates, millimetres and the change from 100% to 200% zoom. The
published View interface options scenario passed all seven steps.
[Part telemetry](part-telemetry.png) · [Scenario report](view-options.json)
· [Published input hashes](current-inputs.sha256)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`66480baec` with the status changes and pending local filesystem and
Ribbon Customization edits.

The GUI library suite passed 825 tests with one ignored catalog test.
Strict Clippy stopped at an existing nested conditional in
`ribbon_panel_measurements.rs`. Clippy passed with that lint allowed.
