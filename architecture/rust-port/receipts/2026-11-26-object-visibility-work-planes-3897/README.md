# User Work Planes in Object Visibility

The User Work Planes and All Workfeatures switches now affect work-plane
display and viewport picking. The focused test created a visible offset
plane and an individually hidden plane. Hiding the category restored the
background pixel at the visible plane's projected center and removed the
plane from click selection. Restoring the master restored the same plane
and its pick target. The individually hidden plane stayed hidden and the
undo list remained unchanged.

All four published checklist scenarios passed 29 steps per desktop.
All six native overlay tests passed per desktop, including undo and redo,
save and reopen, and starting a sketch from a selected work plane.

The build used `ea6c97322`, the [implementation and test changes](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. [Published inputs](current-inputs.sha256) matched before
and after the runs. Strict provenance remains open under #3978.

- [Wayland report](wayland.json), [step arguments](wayland-arguments.json), and [native tests](native-wayland.txt)
- [X11 report](x11.json), [step arguments](x11-arguments.json), and [native tests](native-x11.txt)
