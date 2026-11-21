# Native mouse-wheel zoom diagnostic

The pinned `Change magnification with the mouse wheel` scenario passed all
eight steps under Xvfb and Vulkan Lavapipe. The source was local snapshot
`b576445b98cffd686dcc05aa8c017de2a591c8c8`; the live feature hash matched
before and after the run. The scenario has `needs_fixture`, and native build
provenance remains open (#3978). This is diagnostic evidence, not verified
acceptance.

The host created the solid fixture. X11 XTEST moved the pointer to the block
and sent three wheel steps in each direction. The assertions compared visible
solid bounds, camera orientation, and active part. The three captured frames
were inspected. This does not test a physical mouse, Windows, or macOS.

The first run stopped before wheel input because the inherited XIM setting had
no input provider and the status was not Ready. Its log is retained. The
passing run used `XMODIFIERS=@im=none` for Xlib's local input method.

Go reference tests `TestScrollViewport*` passed at `aa8297d1b`. They cover
wheel direction, zero delta, inversion, and zoom to cursor; they do not prove
native window or visual parity. No application behavior changed in this work.
