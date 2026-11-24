# Grouped X11 model-state activation

Issue #3896. At `ee868a800`, the native suppression test group failed
twice when it ran under Openbox. After the X11 double-click on Primary,
Copy State was still active and Material Browser was open. The native
state-switch test passed by itself on the same build. All 11 suppression
tests also passed together on Xvfb without a window manager.

The test now focuses the native application, processes pointer motion,
and reads the current row position before sending the double-click.
It still uses XTEST for both clicks and checks each active member,
the solid's visibility, and the extrusion's suppression indicator.

Two grouped runs under Openbox each passed all 11 tests. Each used its own
Xvfb display and ran the tests sequentially with Vulkan Lavapipe.
