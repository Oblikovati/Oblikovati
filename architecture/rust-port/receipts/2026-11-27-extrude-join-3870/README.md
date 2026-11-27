# Extrude Join adds a boss to one solid

Scenario 13 starts with a rectangular solid and a diameter-3 mm circle on
its upper plane. Selecting Join with Default direction and Distance A 8 mm
shows the shaded replacement preview. The test measures the cap diameter
and height through the rendered geometry and checks that the current
document generation and body key still describe the original solid.

OK creates Extrusion2 and leaves one eight-face body. The circular sketch
is consumed and the shaded boss remains visible. Undo removes Extrusion2,
restores the six-face base and makes the original circular sketch visible.

| Scenario source | Wayland | X11 |
| --- | --- | --- |
| Published fixture and feature | [JUnit](wayland/current-13.xml) | [JUnit](x11/current-13.xml) |
| Pinned feature at `b576445b9` | [Cucumber](wayland/pinned-13.json) | [Cucumber](x11/pinned-13.json) |

The pinned wording calls the restored circle Sketch2. The published native
package calls that circle Sketch1. Both executions check the circle in the
published package. The fixture, feature file, and command icon are recorded
in [published-inputs.sha256](published-inputs.sha256).
