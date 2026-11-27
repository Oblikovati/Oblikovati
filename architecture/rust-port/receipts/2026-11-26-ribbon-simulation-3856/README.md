# Simulation visibility from the minimized ribbon

Simulation starts visible on the 3D Model ribbon. The published scenario
hid the panel through Show Panels, opened a temporary ribbon without
it, then restored it through the same menu. Double-clicking the tab
showed the full ribbon and Simulation. The disabled Stress Analysis button
indicates that the analysis backend is unavailable.

The 17-step scenario passed on Wayland and X11. The separate minimized
Parameters scenario checks that showing an existing hidden panel
works through the same controls. Compact and side-docked selector cases
also passed on both backends. The GUI keyboard regression verifies
that focus wraps from Sketch to Simulation in the current panel catalog.

The full Wayland run passed 45 of 55 executions and 540 steps in the
passing cases. Two failures use an X11-only View tab click. Three more
stop at assertions about Modify or Parameters captions and expansion.
Three require additional panel inventory, and two require Shell previews.

The build used revision `c5c1a0453`, this change, and existing local GUI
filesystem edits. Both runs used Xvfb and Vulkan Lavapipe. Wayland used
nested Weston 13 and X11 used Openbox, with input sent through the native
window's frame loop. Input hashes matched before and after verification.
Strict provenance remains open under #3978.

- [Wayland report](wayland.json) and [X11 report](x11.json)
- [Minimized Parameters on Wayland](parameters-wayland.json) and [X11](parameters-x11.json)
- [Full Wayland visibility report](full-wayland.json) and [suite input hashes](full-inputs.sha256)
- [Input hashes](inputs.sha256) and [test results](test-results.txt)
