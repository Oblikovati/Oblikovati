# Loft entry and section selection

The published `loft-01` scenario opens Loft in an empty Part, checks the
geometry tip and Help, and dismisses the dialog with OK. In `loft-02`, two
viewport clicks pick the center and circumference of the same sketch
circle. The draft keeps the exact point and circle identities in separate
rows, Point Sketch1 and Sketch1.

| Backend | Empty Part | Point and circle |
| --- | --- | --- |
| Wayland on COSMIC | [7 steps](wayland/loft-01/junit.xml) | [4 steps](wayland/loft-02/junit.xml) |
| X11 under Xvfb and Openbox | [7 steps](x11/loft-01/junit.xml) | [4 steps](x11/loft-02/junit.xml) |

The [input hashes](current-inputs.sha256) identify the published source and
both native fixtures.
