# Use a detached drawing and its tab menu on Wayland

The context-menu and shared-controls scenarios passed all 18 steps on
Wayland and X11. Right-clicking the detached drawing tab showed Close,
Close Tabs, Close All, Save, and Properties, with the Close and Save
shortcuts. Escape dismissed the menu and kept the drawing window open.

Clicking the detached sheet switched the main browser and ribbon to drawing
context. Selecting the Part tab restored the Part browser and ribbon while
the detached drawing remained mapped.

Wayland used the desktop shell in fullscreen nested Weston 13 on a private
2048 × 1200 Xvfb screen. The test targeted the detached toplevel by title and
sent pointer and keyboard events through the compositor. X11 used Openbox
and XTest. Both ran with Vulkan Lavapipe.

The verification build was based on `97f24d2a0` with the test changes and
existing local GUI filesystem edits. The fixture and feature hashes matched
after verification. The native Wayland document-tab shortcut regression
and focused Clippy also passed.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Wayland input trace](wayland-input.txt) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
