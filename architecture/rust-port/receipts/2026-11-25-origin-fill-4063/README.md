# Origin-plane fill in the native viewport

The picker calibrates the authored opacity against the active viewport
backdrop. Dear ImGui supplies sRGB colors and the renderer blends in
linear light. The picker adjusts the fill alpha before sharing it across
overlapping sheets. The calibration uses the channel with the largest
color difference, so individual channels and model surfaces can vary.

The Front-view test samples the viewport before and after opening the
picker. In Dark, the red channel moved from 23 to 103 before the change,
a 36% shift toward amber. It now moves from 23 to 54, about 14%.
In Light, the blue channel moves from 247 to 218, about 12%.
[Before](before-dark.png) · [Dark](dark.png) · [Light](light.png)
· [Oblique view](oblique.png)

Five native origin checks passed in the serial rerun. An earlier run
differed by one RGB byte in the exact ViewCube face comparison. That
comparison passed in isolation and in the serial rerun. The Extrude
shortcut check and six GUI tests passed.

The pinned sketch creation and cancellation scenarios passed nine
steps. [Creation](pinned-0.json) · [Cancellation](pinned-1.json)
· [Pinned source](pinned-source.json)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`791cf35ef` with the fill change and pending local filesystem and Ribbon
Customization edits.
