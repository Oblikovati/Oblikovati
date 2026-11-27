# Move the main window from the application bar

The published move scenario opens the saved Part and drags a blank area of
the application bar. The Wayland test sends the pointer drag through
fullscreen Weston on a private Xvfb display. Weston delivers the native
Wayland press and seat serial. Its scene graph records the application
position and size before and after the drag.

The baseline `f9c41c440` searched for an X11 Oblikovati window after the
Part opened under Wayland. The first backend-aware run moved the window
and exposed ten identical move requests for one press. The fix disarms
the application-bar gesture after handing it to the native window manager.
A GUI regression verifies that held input emits a single Move and a
separate press can start another drag.

All six published steps passed on Wayland and X11. The Wayland trace
records one move request for the compositor-provided press serial.
The window moved 72 pixels horizontally, kept its size, and left
the same Part and solid visible. The seven-step Wayland double-click
scenario also passed after the fix.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Double-click report](double-click.json)

Both native runs used Vulkan Lavapipe and the published Part fixture.
Wayland used Weston 13 with the X11 backend, its desktop shell, and
Pixman. X11 used Xvfb, Openbox, and tint2.
[Baseline report](baseline.json) · [Protocol and geometry](test-results.txt) · [Input hashes](inputs.sha256)
