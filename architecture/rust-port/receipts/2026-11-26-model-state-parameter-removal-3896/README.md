# Cancel removal of a Model State parameter column

The published scenario passed all 13 steps on Wayland and X11. Removing
the chosen d0 parameter hid its column and enabled OK. Cancelling and
reopening restored the column, with the values for Primary, Capture
State, and Copy State still at 1 in, 1 in, and 2 in.

The build used `5c9dbce68`, the [test source hash](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after both runs. Strict build and runtime verification
remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
