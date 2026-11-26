# Distinct Model State parameter values

The published scenario passed all 21 steps on Wayland and X11. Copy
State kept its accepted 2 in value after the table closed and reopened.
Activating Primary refreshed the parameter tree to 1 in while the
table retained the separate values for Primary, Capture State, and
Copy State.

The test reused the browser activation helper and asserted that Primary
became active before checking the parameter tree. The build used
`dba3897a7`, the [test change](test-input.sha256), and the existing local
GUI filesystem test-module visibility edit. Wayland used nested Weston 13
and X11 used Openbox. Both used Xvfb and Vulkan Lavapipe.
The [published input hashes](current-inputs.sha256) matched before and
after both runs. Strict build and runtime verification remains open
under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
