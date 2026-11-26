# Ribbon Panel Visibility on Wayland

The five rerun scenarios passed all 61 steps on Wayland and X11. They
cover switching tabs and parts, restoring Modify with its expansion
closed, dismissing the side selector and expansion with Escape, and
showing Parameters between Format and Exit. The View-tab step uses
desktop pointer input on both backends. The caption checks recognize
the rendered uppercase labels and expansion arrow.

The combined Wayland report updates the five cases from the
[Simulation run](../2026-11-26-ribbon-simulation-3856/README.md).
Fifty of 55 executions passed, covering 601 steps in passing cases.
The five remaining cases require panel inventory or Shell previews.

The rerun used revision `6cf166bc6`, the test changes, and existing
local GUI filesystem edits. Wayland used nested Weston 13, and X11 used
Openbox. Both ran under Xvfb with Vulkan Lavapipe. Published inputs
matched the prior run. Strict provenance remains open under #3978.

- [Combined Wayland report](full-wayland.json) and [X11 cross-check](targeted-x11.json)
- [Input hashes](inputs.sha256) and [test results](test-results.txt)
