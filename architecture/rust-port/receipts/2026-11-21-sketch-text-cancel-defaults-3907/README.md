# Sketch Text defaults after Cancel

Pinned `text-05` passed all six steps at application commit
`78461cabde7c24bd1daa9eac79d8f259bec23d40`. The clean scenario snapshot was
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`.

The fixture sets a blank Tahoma preset as the new-text default within the
session. The native test types `a`, changes its height to 6.10 mm, cancels
the draft, and starts Text at another point. The new editor shows an empty
content area, Tahoma, 3.05 mm height, and 100 percent width. The sketch has
no committed Text. The companion native test passes with DejaVu Sans.

The Linux run used Xvfb, Vulkan Lavapipe, and an isolated Fontconfig file.
Tahoma came from `IELPKTH.CAB` with SHA-256
`c1be3fb8f0042570be76ec6daa03a99142c88367c1bc810240b85827c715961a`.
The Go reference tests for placement, style and blank input passed at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The Rust run used API v0.165.0
and Martensite v0.13.1. [Microsoft Tahoma family](https://learn.microsoft.com/en-us/typography/font-list/tahoma)
[Winetricks archive checksum](https://github.com/Winetricks/winetricks/blob/master/src/winetricks).

The report supplies R0 baseline evidence. Strict verification remains
open under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
