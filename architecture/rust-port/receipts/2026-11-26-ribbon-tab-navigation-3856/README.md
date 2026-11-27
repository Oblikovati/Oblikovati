# Ribbon tab navigation on Wayland and X11

Seven previously failing Wayland cases passed. The drawing's Place Views
ribbon cycled through all four display modes, and the Part scenarios
covered double-click activation, switching temporary commands, and wheel
navigation. The wheel boundary case also passed on both backends. The
eight focused scenarios completed 89 steps on each backend.

The checks read the rendered underline within the tab strip. Consecutive
double-click gestures are spaced beyond the double-click interval. The
existing X11 native wheel test also passed with the updated selection
check.

The full amended Wayland feature passed 43 of 52 scenarios and 488 steps
in passing scenarios. The remaining cases cover Options panel dismissal,
sketch and Home context transitions, and three scenarios with missing
bindings.

The test builds used application revision `fe7e8802e`, the test change in
`09b7669bb`, and existing local GUI filesystem edits. The published
inputs matched their hashes before and after. Wayland used nested
Weston 13 and X11 used Openbox, both on Xvfb with Vulkan Lavapipe.
Strict build and runtime provenance remains open under #3978.

- [Full Wayland report](wayland.json) and [arguments](wayland-arguments.json)
- [Focused X11 report](x11.json) and [arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)
