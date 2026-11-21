# Sketch Text fraction history

Pinned `text-29` passed all six steps on Linux X11 under Xvfb with Vulkan
Lavapipe, API v0.165.0, and Martensite v0.13.1. The test ran from staged
tree `e3d3040d3daed8a2bab0703be998d3c0c8686dd7`, which matches
application commit `4344c4c89a9f48dcb87eaec26db0f16fa163d2f8`.
The clean scenario snapshot was `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The fixture constrains a midpoint line to 10 mm, links one Text to `d0`,
and creates a blue `AB⌀` label in a sketch with 35 dimensions needed.
After replacing that label with a stacked fraction, Undo restores the
diameter label. Redo restores the blue fraction at the same text ID and
anchor. The status shows 35 dimensions needed and the separate linked
Text still displays `10.000`.

The companion native test drives Ctrl+Z and Ctrl+Y through X11. It checks
that Undo changes the displayed label pixels and Redo restores the exact
fraction pixels. The Go glyph and typed Sketch Text tests passed at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The report supplies R0 baseline evidence. Strict verification remains
open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
