# Start an extrusion at a parallel work plane

The published scenario uses the rectangular Sketch1 at Z = 0 and
Work Plane1 at Z = 3 mm. Selecting Reference Plane under From and
entering 8 mm for Distance A produces a start cap at 3 mm and a far
cap at 11 mm. The preview leaves the feature history unchanged. All
eight steps passed on Wayland and X11. [Input checksums](published-inputs.sha256),
[Wayland](wayland.xml), [X11](x11.xml).

The native commit and reopen test uses a byte-for-byte copy of the same
fixture. It checks the two caps and the authored Work Plane1 reference
after reopening the saved Part. The model recipe test changes the plane
offset and extrusion depth independently and checks the two resolved
inputs without rewriting the recipe. [Wayland reopen](wayland-reopen.txt),
[X11 reopen](x11-reopen.txt).

The existing typed command uses `extent: "distance-from-face"` and
its `toFace` field for the single reference. The host treats that plane
as the start, as specified in the published scenario. Native Part
program version 42 stores the plane key and the authored length.
