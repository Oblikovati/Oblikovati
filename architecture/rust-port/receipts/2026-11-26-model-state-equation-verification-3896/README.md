# Report an unevaluable Model State equation

The published scenario passed all 18 steps on Wayland and X11. Entering
an unknown parameter turned the cell yellow. Verify preserved that
highlight. OK presented the affected parameter and equation in a
warning, which the user could dismiss to return to the draft. Cancelling
the table discarded the entry.

The build used `5893a59b9`, the [test source hash](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb
with Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after both runs. Strict build and runtime verification
remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
