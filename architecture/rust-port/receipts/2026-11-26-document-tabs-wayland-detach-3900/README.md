# Detach a drawing tab in the Wayland desktop

The nine published steps passed on Wayland and X11. Dragging the drawing tab
out of the strip opened a native window with its own tab and visible sheet.
The main window kept a Part open. Selecting the other Part left the drawing
mapped in its separate window.

The Wayland run used fullscreen nested Weston 13 with its desktop shell.
The outer Xvfb screen and Weston output both measured 2048 × 1200 at scale 1.
Pointer motion and button events went through the compositor. The scene
graph recorded the drawing on surface 526 and the main window on surface 5,
which switched from PartOne to PartTwo. The X11 run used Xvfb, Openbox, and
XTest. Both used Vulkan Lavapipe.

The build used application revision `b4feb6a11`, the test changes, and
existing local GUI filesystem edits. The feature and fixture hashes matched
before and after verification. The six-step Wayland application-bar move
regression and focused Clippy also passed.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Wayland toplevels](wayland-toplevels.txt) · [Wayland pointer events](wayland-pointer.txt) · [Input hashes](inputs.sha256)
