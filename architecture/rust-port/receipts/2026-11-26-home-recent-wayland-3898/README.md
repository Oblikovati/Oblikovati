# Home Recent Documents on Wayland and X11

Thirty-four of the 35 published scenarios passed on both backends. Each
combined report contains the full run with the focused rerun substituted
for the row-hover case. The old hover assertion sampled the document-type
icon. The corrected assertion samples the space between Name and Location,
where the highlighted row changes the framebuffer color. The rerun also
verified that the row's actions menu opens.

The published no-document startup scenario still expects an empty Model
browser and asset dock. Home now hides those docks under GAP-20. The
separate startup test passed on Wayland and X11. It verified full-width Home
and the Model and asset docks after choosing New Part.

Application code was at `10905c923` with existing local GUI filesystem edits.
The focused checks also included the revised test code. Fixtures and feature
support hashes matched before and after the full run. Wayland ran on nested
Weston 13 and X11 ran on Xvfb with Openbox. Both used Vulkan Lavapipe.
The X11 startup test used a native click and the Wayland test used frame-loop input.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Startup capture](startup-home.png) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
