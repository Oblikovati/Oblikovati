# Sweep entry and history

The native Part Create ribbon runs the published `sweep-01` and `sweep-02`
scenarios against disposable copies of their fixtures. The first checks
the empty Part tip and Help. The second checks the chosen profile and path,
teal viewport fill, enabled OK, the Sweep1 and Solid Bodies browser rows,
Undo restoring two visible sketches, and Redo restoring the solid with
three faces.

| Backend | Empty Part | Straight-path Sweep |
| --- | --- | --- |
| Wayland on COSMIC | [5 steps](wayland/sweep-01/junit.xml) | [11 steps](wayland/sweep-02/junit.xml) |
| X11 under Xvfb and Lavapipe | [5 steps](x11/sweep-01/junit.xml) | [11 steps](x11/sweep-02/junit.xml) |

The build used Martensite v0.14.2. The [input hashes](current-inputs.sha256)
identify the published feature, fixtures, and feature artwork.

The [Wayland capture](wayland/sweep-preview.png) shows the shipped Sweep
icon, the profile and path counts, and the certified teal preview.
