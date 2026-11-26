# Create a feature in one Model State

The published scenario passed all 12 steps on Wayland and X11. The
States button listed the new extrusion as suppressed in Primary and
Capture State and computed in Copy State. Selecting Primary in the
Model State branch removed the sampled face and dimmed and struck
through the extrusion name. Returning to Copy State restored the
visible solid and the regular row.

The build used `c8572c034`, the [test source hash](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb
with Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after both runs. Strict build and runtime verification
remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
