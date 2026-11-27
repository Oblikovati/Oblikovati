# 3D Print and keyboard focus in Show Panels

The 3D Print panel is visible by default and is the last row in the 3D
Model checklist. The published keyboard scenario passed all 11 steps on
Wayland and X11. It verified which checkbox receives focus when wrapping
from Sketch to 3D Print and back, and compared each checkbox before and
after keyboard navigation. A standalone desktop test passed on both
backends. The panel's Export STL control is disabled.

The full Wayland run passed 51 of 55 cases, with 612 steps in the passing
cases. Two cases still require the complete Manage and 3D Model panel
inventories, and two require Shell previews.

The build used revision `ece3bda4e`, this change, and existing local GUI
filesystem edits. Wayland ran on nested Weston 13 and X11 on Openbox.
Both used Xvfb and Vulkan Lavapipe, with input through the native window's
frame loop. The published input hashes matched before and after the run.
Strict provenance remains open under #3978.

- [Wayland keyboard scenario](wayland.json) and [X11 keyboard scenario](x11.json)
- [Full Wayland report](full-wayland.json)
- [Input hashes](inputs.sha256) and [test results](test-results.txt)
