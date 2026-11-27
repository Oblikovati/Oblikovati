# Current ViewCube hover and orientation scenarios

The published ViewCube feature passed all 26 scenarios and 284 steps on
both Wayland and X11. Native pointer movement highlighted each face's
center separately and lit every visible side of the shared Top-Front edge
and Top-Front-Right corner. Moving across the seam preserved the highlight
pixel for pixel. Leaving the control restored its idle appearance. Camera
and model scene generations stayed fixed throughout hover.

The remaining scenarios also exercised animated turns, projection, compass
and placement options, size, opacity, and selection lock. The Wayland run
used nested Weston 13 and the X11 run used Openbox. Both ran on Xvfb with
Vulkan Lavapipe. The published scenario, supporting files, and Part fixture
matched their hashes before and after both runs.

The GUI library suite passed 839 tests. Its external design catalog test
was ignored by default. GUI and Head passed Clippy with all targets and
features enabled.

The test binary used the ViewCube implementation in `9af470f54` and
existing local GUI filesystem edits. Issue #3978 tracks strict build and
runtime provenance.

- [Wayland report](wayland-cucumber.json) and [arguments](wayland-scenario-arguments.json)
- [X11 report](x11-cucumber.json) and [arguments](x11-scenario-arguments.json)
- [Published input hashes](inputs.sha256)
