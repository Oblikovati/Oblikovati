# Measure the Extrude Intersect result

Scenario 17 creates the common solid, then undoes and redoes the
intersection. In the Measure tool the native test picks the top cap and
records its 7.069 mm² area. It then picks the cylindrical side and checks
the 3.000 mm diameter and 1.500 mm radius. The radius comes from the
analytic cylindrical surface bound to the picked face key.

| Scenario source | Wayland | X11 |
| --- | --- | --- |
| Published fixture and feature | [JUnit](wayland/current-17.xml) | [JUnit](x11/current-17.xml) |
| Pinned feature at `b576445b9` | [Cucumber](wayland/pinned-17.json) | [Cucumber](x11/pinned-17.json) |

Both executions use the native package identified by the
[Extrude Join receipt](../2026-11-27-extrude-join-3870/README.md).
