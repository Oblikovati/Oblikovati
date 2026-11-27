# Current Ribbon Command Availability on Wayland and X11

All five published scenarios and 39 steps passed on each desktop. The
checks cover the Sketch ribbon before and during Sketch1 editing,
an idle saved Assembly, and the Place Views and Annotate groups above
the Drawing sheet. The drawing checks now locate the uppercase group
captions used by the ribbon.

The build used `22270f57c` with the [test source change](test-input.sha256)
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after both runs. Strict build and runtime verification
remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
