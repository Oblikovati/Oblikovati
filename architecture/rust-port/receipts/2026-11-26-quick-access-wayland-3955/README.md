# Current Quick Access toolbar on Wayland and X11

All 13 published declarations passed on both desktops, with 25 executed
cases and 268 steps per backend. The suite includes the 13 native hover
examples, command and separator placement, per-document shortcuts,
Extrude invocation, and menu dismissal without changing check states.

The Extrude tests start with an oblique camera so all three projected
origin planes are readable. A first run started face-on, where two
planes project edge-on. The revised assertion reads each plane
identifier inside the viewport picker. The menu assertion reads the
rendered rows in the Quick Access popup and recognizes iProperties
when glyph decoding drops the leading i. The screenshot showed the
full label.

The build used `a8b8179aa` with the [test source changes](test-input.sha256)
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after both runs. Strict build and runtime verification
remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
