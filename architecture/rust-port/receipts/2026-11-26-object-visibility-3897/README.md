# Object Visibility checklist

All four published scenarios passed, with 29 steps on each of Wayland
and X11. The checklist opens below its ribbon control, stays open while
switches change, and leaves the Model Browser and solid visible. All
Workfeatures affects the origin, user work-feature, and UCS checks.
An individual switch leaves the master and other checks alone.

A focused test hid and restored a rendered origin axis and sketch
dimensions. The solid remained visible. The existing View ribbon
panel hide and restore scenario passed 12 steps on both desktops.
Additional geometry fixtures and renderer integration are needed for
the other categories.

The build used `e832efe38` with the [implementation and test changes](test-input.sha256)
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb
and Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after the runs. Strict build and runtime verification
remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Wayland screenshot](checklist-wayland.png)
- View ribbon regression on [Wayland](view-ribbon-wayland.json) and [X11](view-ribbon-x11.json)
