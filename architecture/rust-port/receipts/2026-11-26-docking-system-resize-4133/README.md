# Docked asset controls during native resize

All 14 amended Docking System scenarios passed on Wayland and X11,
with 140 steps on each backend. The new scenario put Model Browser on
the left and tabbed Material Browser with Appearance Browser on the
right. Native corner drags narrowed the window to about 960 × 650
and then 792 × 580.

At the first width, the docks shared the space above their minimums
in proportion to their preferred sizes. At the narrower width, the
viewport kept at least 55% of the window width within splitter rounding,
and the solid stayed visible. Both asset selectors kept their captions
on the left and shortened the selected values before the arrows.
Framebuffer samples confirmed inactive title contrast and the borders
around the black and white swatches in Dark and Light. Both theme
captures were visually inspected on Wayland.

The native build used the product changes in `8c298fe64` and existing
local GUI filesystem edits. The earlier scenarios ran before the final
contrast test assertion was adjusted. Wayland used nested Weston 13,
and X11 used Openbox, both with Xvfb and Vulkan Lavapipe. The standalone
X11 restart case was rerun at 1440 × 900, as its driver requires. Published
input hashes matched before and after the runs. Issue #3978 tracks strict
build and runtime provenance.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
