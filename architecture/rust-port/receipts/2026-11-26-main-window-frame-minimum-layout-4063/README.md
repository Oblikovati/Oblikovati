# Main Window Frame at minimum size

All 11 current declarations passed separately on Wayland and X11.
Each backend ran 20 cases and 171 steps. At 640 × 480 both side docks
narrowed, the active Part tab kept its accent underline, and the ribbon
fit all compact groups. Opening the trailing 3D Print group exposed
Export STL. The floating Shell panel stayed inside the viewport and
Cancel closed it. The application-bar and desktop Close actions both
opened the save-changes prompt for an edited Part.

The Wayland cases used Weston, with private Labwc for desktop Close
and Labwc with Waybar for minimization and reactivation. The minimum
drag used a private Weston 13 test build with the xdg-toplevel
minimum-size getter connected to its shell.
[Weston test patch](../2026-11-26-wayland-minimum-frame-4063/weston-13-min-size.patch)

X11 used Xvfb,
Openbox and tint2. Both accessibility runs used a private AT-SPI bus.
Rendering used Vulkan Lavapipe on both backends.

The X11 suite ran through
`crates/head/tests/accessibility_atspi/frame_run.sh` with
`OBK_BDD_ROOT` pointing to the published source. Wayland used that
script with `--wayland` for AT-SPI and
`crates/head/tests/application/main-window-frame/labwc_run.sh` for
the two Labwc cases. Other Weston cases selected their declaration
with `OBK_BDD_SCENARIO`. The private desktop scripts invoked the
prebuilt head library test executable directly. All runs used
`application_scenarios_use_the_native_window`.

The feature, support files, and fixture hashes matched before and after
execution. The test build included existing local GUI filesystem edits.
Strict build and runtime provenance remains open under #3978.

- [Wayland report](wayland.json) and [resolved steps](wayland-arguments.json)
- [X11 report](x11.json) and [resolved steps](x11-arguments.json)
- [Parsed inventory](scenario-inventory.json) and [input hashes](current-inputs.sha256)
