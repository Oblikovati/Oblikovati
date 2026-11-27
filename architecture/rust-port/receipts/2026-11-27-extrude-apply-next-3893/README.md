# Extrude Apply and Create New

The published Properties scenario opens the rectangle in
`revolve-profile.opd` and previews a Flipped extrusion at 8 mm.
Apply and Create New commits one feature with caps at -8 mm and
0 mm. The next draft retains the distance and direction and prompts
for a sketch and profile. OK stays disabled while the next draft waits
for geometry. Cancel closes the draft and leaves Extrusion1 in place.
[Published input checksums](published-inputs.sha256).

The next draft retains operation and termination choices and clears
the source sketch, profile index, and selected target references.
Numeric validation uses the document generation after Apply so the
next request resolves its current parameter values.

All six published steps passed on Wayland and X11. The native
regression for preview visibility and the fresh draft also passed on
both desktops. All six Extrude ribbon integration tests passed.
[Wayland scenario](wayland.xml), [Wayland regression](wayland-native.txt),
[X11 scenario](x11.xml), [X11 regression](x11-native.txt).
