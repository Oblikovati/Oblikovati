# Available command list keyboard navigation

The published sequence starts with 2D Sketches selected in ascending All
Commands order. End scrolls to and highlights Zoom Window. Home returns
to the separator. Z scrolls to and highlights Z Parallel. The Part Tools
target stays empty and clean, Apply stays disabled, and Close dismisses
Customize.

The binding uses the existing Home, End, and Z input steps. The empty-target
check now reads the target list directly, so it works when the source list
has scrolled away from its separator. The Macros scenario still checks that
the source initially holds one separator and that a copy can be staged.

The keyboard sequence passed all 17 steps on Wayland and X11. The Macros
separator scenario passed all 15 steps on each backend.

[Wayland keyboard report](wayland.xml) · [X11 keyboard report](x11.xml) · [Published inputs](current-inputs.sha256)

The build used base `78b705508a92029050c174f814934d5dc2fbeeff`, this change,
and existing local GUI filesystem and test edits. Wayland used headless Weston
with Pixman and X11 used Xvfb with Openbox. Both used Vulkan Lavapipe and
frame-loop input.
