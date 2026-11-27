# Ribbon context while editing an existing sketch

The existing-sketch scenario passed all 12 steps on Wayland and X11.
Opening the extrusion's sketch selected Sketch without expanding the
ribbon. Selecting Tools exposed Finish in the temporary Exit group.
Finishing restored the shaded Part and left Tools selected in tabs-only
mode. The test clicked the feature's expansion arrow to the left of its
icon and checked selection from the rendered ribbon underline.

The full Wayland run passed 49 of 52 scenarios and 563 steps in passing
cases. Two remaining scenarios require a Shell preview. The split-button,
single-command sizing, and combo-label scenario still needs coverage.

The native build used commit `cd2a49ada` and existing local GUI filesystem
edits. Wayland ran under nested Weston 13, and X11 used Openbox. Both used
Xvfb and Vulkan Lavapipe. Published scenario, support, and fixture hashes
matched before and after the runs. Strict build and runtime provenance
remains open under #3978.

- [Full Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 sketch context report](x11-sketch-context.json)
- [Published input hashes](inputs.sha256)
