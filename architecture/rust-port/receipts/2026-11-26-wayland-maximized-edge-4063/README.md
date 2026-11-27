# Keep the maximized window bounds during an edge drag

The published scenario opens a native Part, maximizes the frame, then drags
from the midpoint of the left edge toward the center. It compares the
window's position and size before and after the drag, checks that the
window remains maximized, and checks the Restore tooltip.

The Wayland run uses stock Weston 13 nested fullscreen on private Xvfb
with Openbox. XTest sends pointer motion and button events to Weston.
Its scene graph reports the window bounds, and the Wayland protocol
trace records delivery of the drag to Oblikovati. The X11 run uses Xvfb
with Openbox. Both render with Vulkan Lavapipe.

At baseline `159c19d98`, the test opened the Part on Wayland and then
searched for an X11 Oblikovati window. The verification build used the
backend-aware steps and included the existing local GUI filesystem edits.
[Baseline report](wayland-before.json) · [Input hashes](inputs.sha256)

All four steps passed on both backends. The Wayland drag reached
Oblikovati and issued zero resize requests. Weston's scene graph
reported the same position and size before and after the drag.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Native trace](test-results.txt)
