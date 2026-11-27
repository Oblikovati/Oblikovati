# Transfer consecutive available ribbon commands

The All Commands catalog lists 3D PDF and 3D PDF Export after 3D Annotations.
Each has a distinct settings identifier. Both can be staged and saved in
a custom ribbon panel, where their buttons are disabled and the tooltip
reports their availability.

The published scenario uses the native Part and Normal ribbon profile. It
selects 2D Sketches once, then clicks Add three times. The first two clicks
stage 2D Sketches and 3D Annotations and select 3D PDF. The third click
stages 3D PDF and selects 3D PDF Export. The test checks the rendered selection,
the pending marker, and the dialog's dismissal with Cancel.

The baseline used `7a723545c` with the scenario steps and the existing
local GUI filesystem edits. It staged the first two commands and then failed
because 3D PDF was absent. The verification build also included the local
GUI edits. Wayland used private nested Weston 13, and X11 used Xvfb with
Openbox. Both ran with frame-loop input and Vulkan Lavapipe.
[Baseline report](wayland-before.json) · [Input hashes](inputs.sha256)

All 15 published steps passed on Wayland and X11. The 58 Ribbon Customization
unit tests passed, including both identifiers in the settings round trip.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Run details](test-results.txt)
