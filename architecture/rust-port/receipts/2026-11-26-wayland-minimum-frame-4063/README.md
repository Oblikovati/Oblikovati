# Keep the application controls reachable at minimum size

The published scenario passed all six steps under X11 and under a private
Weston 13 build with its xdg-toplevel minimum-size getter connected. The
native drag stopped at 640 × 480. The frame buttons maximized, restored, and
minimized the window. Super+Tab reactivated it on Weston and the tint2 taskbar
reactivated it on X11. The same saved Part remained active, clean, and visible.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Run details](test-results.txt)

When Weston owns the native resize grab, its button release may reach only the
compositor. Oblikovati used to consume the next frame button's release, too.
The regression reproduced with real Wayland pointer input and worked after
resetting the resize flag on a new client-area press.
[Before the pointer fix](resize-click-before-fix.json)

The separate close test drags directly to 640 × 480, verifies the original dock
layout after expanding, then resizes to the minimum and clicks Close. It checks
that the clean application exits without a confirmation. This test uses stock
Weston and X11.

Stock Weston 13 accepted `xdg_toplevel.set_min_size(640, 480)` and still sent
a 102 × 102 configure when the drag went past the minimum. Its xdg-toplevel
implementation stores the hint, but leaves the getter unset for the shell.
The [Weston 13 fixture patch](weston-13-min-size.patch) connects that getter to
Weston's existing resize clamp. Apply it with `patch -p1` to the Weston 13.0.0
release source and keep the build private to the test. The source archive SHA-256 is
`52ff1d4aa2394a2e416c85a338b627ce97fa71d43eb762fd4aaf145d36fc795a`.
[Stock Weston report](weston-stock.json) · [Weston 13 xdg-shell source](https://github.com/wayland-mirror/weston/blob/13.0.0/libweston/desktop/xdg-shell.c)

The baseline at `c2fcd97ea` opened the saved Part and then searched for an X11
Oblikovati window on the Wayland run. Both backends used Vulkan Lavapipe. The
scenario and native fixture are identified by content hashes because the
published source directory has no Git revision. The verification build also
included the existing local GUI filesystem edits.
[Baseline report](baseline.json) · [Input hashes](inputs.sha256)
