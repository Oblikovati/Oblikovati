# Model Browser context menus and Open File Location

The saved Part root offers Open File Location. The desktop owner passes the
saved file's parent to the file manager. The command is disabled for an
untitled Part. The native menu tests check the exact directory passed to the
desktop boundary and confirm the saved Part remains active. A separate unit
test verifies a directory with spaces is passed as one argument and checks
rejection of relative and missing directories.

Eight published Model Browser scenarios passed 94 steps on Wayland. They
cover the Part root, extrusion, origin plane, axis, center point, Origin
branch collapse, and two Relationships cases. The axis screenshot assertion
now checks the selected datum's blue pixels. The run also checks that the
visible axis remains after deselection and disappears after toggling Visibility.
[Wayland report](cucumber.json) · [Axis baseline](axis-baseline.json) · [Input hashes](inputs.sha256)

Fourteen focused tests passed on each Linux backend. Wayland used headless
Weston with Pixman and scripted window events, with one compositor per test.
X11 used Xvfb and Openbox with XTest pointer input. Both used Vulkan Lavapipe.
The saved and untitled root cases also passed on both backends after the
handler switched to the owner's filesystem path.

The runs used `628df07d4` with the context menu changes and existing local
GUI file-dialog edits. The published scenario snapshot is unversioned.
Issue #3978 tracks strict build and runtime provenance.
