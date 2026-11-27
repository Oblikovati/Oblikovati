# Native Viewport Pan and Orbit on Wayland

Viewport Pan passed all four published scenarios and 32 steps, and Viewport
Orbit passed all ten cases and 94 steps, on Wayland and X11. Both Orbit
examples of the adjacent-pair outline ran. These runs used the published
saved Part and sent input through the native window's frame loop.

Three separate tests sent pointer and keyboard input through each desktop.
Pan moved the camera eye and target together while keeping scale and up
direction. Its context menu let the user finish panning, switch to Zoom,
and return to the Part menu without editing the document. Orbit rotated
the saved Part, then Escape dismissed the guide and left the Part active.
The Wayland trace contains pointer motion, primary and secondary mouse
buttons, and Escape.

The build used application revision `cc6469b35`, the test changes, and
existing local GUI filesystem edits. Wayland used nested Weston 13 with
a 2048 × 1200 Xvfb host. X11 used Xvfb and Openbox. Both used Vulkan
Lavapipe. Published input hashes matched before and after the runs.
Strict build and runtime provenance remains open under #3978.

- Pan [Wayland](pan-wayland.json) and [X11](pan-x11.json)
- Orbit [Wayland](orbit-wayland.json) and [X11](orbit-x11.json)
- [Direct Wayland input trace](wayland-input.txt)
- [Published input hashes](inputs.sha256) and [bundled fixture hashes](bundled-fixtures.sha256)
- [Test results](test-results.txt)
