# Switch document tabs with native Wayland shortcuts

Five published scenarios passed all 40 steps on each backend. Ctrl+Tab
and Ctrl+Shift+Tab switch in both directions, keep neighboring Parts open,
preserve sketch editing across a visit to Home, and wrap between Home
and the last numbered window. The direct switching regression also passed
on both backends.

The Wayland test focuses the private, fullscreen Weston output and sends
key presses to that compositor. The application receives the Control,
Shift, and Tab events through its Wayland seat. The trace records event
delivery in the adjacent-tabs scenario. The X11 run sent the same chords
directly to the application window using XTest. The existing detached
drawing scenario also passed on X11.

The test build was based on `494f35d0c` with the native keyboard test
changes and existing local GUI filesystem edits. Both runs used Vulkan
Lavapipe. The fixture and feature hashes matched after verification.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Wayland keyboard trace](wayland-keyboard-trace.txt) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
