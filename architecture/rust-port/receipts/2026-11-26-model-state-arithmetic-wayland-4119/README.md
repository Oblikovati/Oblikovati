# Retain a model-state parameter expression

The published Model Browser scenario passed 14 steps on each backend. It
opened the saved blank Part, sketched an undimensioned rectangle, extruded
it by 1 in, and created Capture State and Copy State through the Model Browser.

Reopening the table showed `1 in + 0.5 in` in both the Copy State cell and
the active parameter tree. Primary and Capture State retained `1 in`.
The Parameters panel showed the 1.5 in result and the host reported the
corresponding model value of 3.81 cm.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Run details](test-results.txt)

At baseline `e34d04e93`, the fixture opened on Wayland, then the rectangle
setup searched for an X11 Oblikovati window. The updated steps use the active
backend for sketch and table input. Wayland uses the frame-loop input helpers
on private nested Weston 13. X11 uses Xvfb and Openbox. Both render with
Vulkan Lavapipe. The verification build also included the existing local GUI
filesystem edits. The published inputs have content hashes.
[Baseline report](wayland-before.json) · [Input hashes](inputs.sha256)

The separate `d1` cases still require the extrusion taper contract tracked
in #4127 and Martensite #463.
