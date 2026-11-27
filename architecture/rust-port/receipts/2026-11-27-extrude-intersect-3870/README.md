# Retain the common region with Extrude Intersect

Scenario 16 starts with the rectangular base and the 3 mm circle.
Intersect switches the default direction to Flipped. The shaded
replacement preview contains the cylindrical common region, and the test
checks its bounds along all three axes before accepting the 8 mm draft.

The browser retains one three-face solid and shows Extrusion2. Undo
restores the six-face base and its visible circular sketch. Redo restores
the cylinder. Reopening the saved Part retains the cylinder dimensions
and the same body key.

| Scenario source | Wayland | X11 |
| --- | --- | --- |
| Published fixture and feature | [JUnit](wayland/current-16.xml) | [JUnit](x11/current-16.xml) |
| Pinned feature at `b576445b9` | [Cucumber](wayland/pinned-16.json) | [Cucumber](x11/pinned-16.json) |

The model test covers the recipe round trip, the required version, and
rejection of simultaneous Cut and Intersect targets. Both native runs
use the package identified by the
[Extrude Join receipt](../2026-11-27-extrude-join-3870/README.md).
