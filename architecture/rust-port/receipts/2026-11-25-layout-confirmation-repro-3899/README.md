# Reproduce populated drawing layout confirmation

The confirmation scenarios select the published layout and click the OK
button in Select Component. The first case checks the proposed Part.
The alternate case opens two real Part files, edits both in memory, and
browses to the alternate file. It checks the file name in the component
field before clicking OK.

The proposed Part reports that it needs an update immediately after the
fixture edit. By the time Select Component handles the click, the same
Part reports `requires_update = false`. The desktop refreshes the active
Part for viewport presentation during the chooser interaction. A query
made only at confirmation cannot recover the state observed when the
chooser opened.

Select Component still disables OK. The two warning assertions establish
where the published cases stop. The cancellation cases remain green.

The build used `02b158f4a` with the scenario bindings in this commit and
existing local filesystem and sketch-marking test edits. Wayland used
headless Weston with frame-loop input. X11 used Xvfb and Openbox.
Both used Vulkan Lavapipe. Strict build and runtime provenance remains
open under #3978.
[Wayland report](wayland-cucumber.json) · [X11 confirmation report](x11-cucumber.json) · [Input hashes](inputs.sha256) · [Alternate component](alternate-component-wayland.png)
