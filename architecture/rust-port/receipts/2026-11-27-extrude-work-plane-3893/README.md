# Extrude to a selected work plane

The published termination outline loads a translated 3 mm radius circle
at Z = -12 mm. On the pinned Martensite v0.14.2 revision `d4635266`,
the standalone kernel probe reproduces incomplete planar-region
certification for this exact placement. The 3 mm radius origin control
and the 2.5 mm radius translated control both certify with the expected
analytic area. [Probe source](region_probe.rs), [probe output](region-probe.txt),
[Martensite #619](https://github.com/Oblikovati/Martensite/issues/619).

The published outline remains incomplete. Work point and vertex
termination are also pending. Both starting files retain their
[published checksums](source.sha256).

## Application control

The native regression opens a byte-for-byte bundled copy of
`face-extension-candidate.opd` and selects its rectangular Sketch2. With
To Face active, it selects XY Plane, checks for a preview cap at Z = 0,
then selects Work Plane1 and checks for a cap at Z = 10 mm. The starting
cap remains at Z = -12 mm. Preview leaves history untouched. OK creates one feature
with a recipe bound to Work Plane1's feature ID.

The Wayland run uses the live compositor at 125% and verifies the
floating Extrude panel leaves the Model browser rows reachable. The X11
run uses Xvfb and Openbox with Lavapipe.

Both runs used scenario snapshot `b576445b`. The existing face-pick
test sends X11-specific input, so it ran on X11. [Wayland result](wayland.txt),
[X11 result](x11.txt).
