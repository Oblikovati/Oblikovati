# Compact Options panel dismissal

Clicking the lower OPTIONS caption closed the temporary panel and
left the compact Tools buttons visible. The scenario passed all eight
steps on Wayland and X11. The test clicked the caption inside the
temporary panel after checking that it sits below Document Settings.

All four compact Options scenarios passed 42 steps on each backend.
They also covered Escape, repeating the same panel button, and
dismissal through the selected Tools tab. The full Wayland feature
passed 44 of 52 scenarios and 496 steps in passing cases.

The test builds used revision `f59ccad24`, the test change in
`2351cd2bc`, and existing local GUI filesystem edits. The published
input hashes matched before and after the runs. Wayland used nested
Weston 13, and X11 used Openbox. Both used Xvfb and Vulkan Lavapipe.
Strict build and runtime provenance remains open under #3978.

- [Full Wayland report](wayland.json) and [arguments](wayland-arguments.json)
- [Focused X11 report](x11.json) and [arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
