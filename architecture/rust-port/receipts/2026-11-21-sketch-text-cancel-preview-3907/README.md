# Cancel a live Sketch Text preview

Pinned `text-06` passed all ten steps on Linux X11 under Xvfb with Vulkan
Lavapipe, API v0.165.0, and Martensite v0.13.1. The staged test tree
`98634ce2c2eea7308854a846fad49d328349c1c0` matches application commit
`e40512b0474f9efa5824837fd156c4a62ffda600`.
The clean scenario snapshot was `b576445b98cffd686dcc05aa8c017de2a591c8c8`.
The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The test creates an XY sketch with real sketch entities and 210 dimensions
needed. A blank preset supplies Tahoma. At (450, 650) in the 1440 by 900
window, typing `a` changes the native viewport pixels at a sampled point on
the glyph. Cancel restores those pixels to the baseline and leaves Text
active. Escape returns the sketch to Ready with equal geometry and 210
dimensions needed.

The pinned run used an isolated Tahoma font from `IELPKTH.CAB` with SHA-256
`c1be3fb8f0042570be76ec6daa03a99142c88367c1bc810240b85827c715961a`.
The companion native test uses DejaVu Sans and drives Escape through X11.
The Go placement, style, and blank input checks passed at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The report supplies R0 baseline evidence. Strict verification remains
open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
