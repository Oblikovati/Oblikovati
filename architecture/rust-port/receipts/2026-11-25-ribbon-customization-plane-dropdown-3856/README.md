# Native Offset from Plane drop-down

The published scenario passed 11 steps. The grouped Plane drop-down now
offers Offset from Plane and opens the same native properties dialog as
the main Plane button. The scenario opened the dialog, verified the
planar face prompt and disabled OK, then canceled with Escape and
verified that the original solid remained visible without a new work plane.

The [earlier frame](plane-menu-before.png) shows the original drop-down
with its constructor choices. The
[native regression results](native-regressions.txt) also passed for
the main Plane button and the group and ungroup controls.

The Linux X11 build used the changes in this commit and existing local
GUI filesystem, opening and saving edits. Runs used Xvfb, Openbox and
Vulkan Lavapipe. The [input hashes](current-inputs.sha256) matched the
isolated documents and preference seed. Issue #3978 tracks strict build
and runtime provenance.
