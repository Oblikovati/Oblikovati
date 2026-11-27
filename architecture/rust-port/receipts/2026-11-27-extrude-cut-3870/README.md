# Flip and replay Extrude Cut

Scenario 15 opens the base-circle Part and selects Cut while the Extrude
direction is Default. The controls switch to Flipped and the certified
replacement preview shows an opening. The native test checks the shaded
preview and probes the rendered upper cap at the circle center and at a
point outside the circle.

Accepting an 8 mm depth records Extrusion2 with one seven-face solid.
Undo restores the six-face base, closes the opening, and shows the source
circle again. Redo restores the one-body cut and its opening.

| Scenario source | Wayland | X11 |
| --- | --- | --- |
| Published fixture and feature | [JUnit](wayland/current-15.xml) | [JUnit](x11/current-15.xml) |
| Pinned feature at `b576445b9` | [Cucumber](wayland/pinned-15.json) | [Cucumber](x11/pinned-15.json) |

The two runs use the same native package as
[Extrude Join](../2026-11-27-extrude-join-3870/README.md).
