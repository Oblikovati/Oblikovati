# Application bar controls through Wayland AT-SPI

The two published application-bar examples passed all 16 steps on
Wayland and all 16 on X11. They cover Home and the saved Part, including
the application and document titles, the Quick Access row, and three
distinct window-control icons. The external AT-SPI client found the
Minimize, Maximize, and Close buttons in order and matched their bounds
to the rendered controls.
[Wayland report](wayland.json) · [X11 report](x11.json)

On Wayland the check uses coordinates relative to the window. X11
uses screen coordinates and verifies zero window-manager frame extents.
Wayland ran under headless Weston with the kiosk shell. The runner used
a private D-Bus session and an AT-SPI registry on each backend. X11 used
Xvfb, Openbox and tint2. Both used Vulkan Lavapipe.

The baseline build `0c23ad2c4` stopped in restored-window setup when it
tried to open an X11 display under Wayland. The passing builds used that
revision with the runner and test changes and existing local GUI
filesystem edits. The published sources and fixture matched before and
after the runs. [Baseline report](baseline.json) · [Input hashes](inputs.sha256)

Run with `crates/head/tests/accessibility_atspi/frame_run.sh --wayland`
on a Wayland compositor. The default runner starts Xvfb for the
X11 feature suite.
