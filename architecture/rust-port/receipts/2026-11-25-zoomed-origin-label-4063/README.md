# Origin picker labels on cropped planes

At 1000% zoom in Front view, all four XY Plane corners are outside the
viewport. The old picker drew the plane but dropped its label. When the
original corner placements are clipped, the picker now computes a label
position inside the visible part of the projected plane. The label and
its hit region use the same projection.
[Before](before.png) · [After](after.png)

The Linux X11 regression reproduced the missing label with the old code,
then clicked the restored XY Plane label to enter sketch edit. It also
verified that edge-on XZ and YZ labels remain suppressed. Three existing
native origin checks and the Extrude shortcut check passed. Five GUI
tests covered the projected footprint, grazing angles, and a plane
outside the viewport.

The pinned Create Sketch and Cancel scenarios passed all nine steps.
The current published versions of those scenarios and the startup-plane
scenario stopped at their first unbound setup step.
[Pinned creation](pinned-0.json) · [Pinned cancellation](pinned-1.json)
· [Pinned revision](pinned-source.json) · [Current creation](current-0.json)
· [Current cancellation](current-1.json) · [Current startup planes](current-2.json)
· [Published input hashes](current-inputs.sha256)

Verification used Linux X11, Xvfb and Vulkan Lavapipe. The build used
`1f5db9d56` with the label change and pending local filesystem and
Ribbon Customization edits.
