# Restore saved Extrude properties

Double-clicking the saved rectangular extrusion opens Edit Feature.
The native test checks the restored profile, sketch-plane start, 20 mm
distance, current body name, and zero taper. Match Shape is checked and
disabled for the saved closed profile. The preview spans 20 mm, and
Cancel retains the original feature and solid.

| Scenario source | Wayland | X11 |
| --- | --- | --- |
| Published extrusion-24 | [JUnit](wayland/current-24.xml) | [JUnit](x11/current-24.xml) |
| Published extrusion-live-01 | [JUnit](wayland/current-live-01.xml) | [JUnit](x11/current-live-01.xml) |
| Pinned extrusion-24 | [Cucumber](wayland/pinned-24.json) | [Cucumber](x11/pinned-24.json) |

The published tests use `extrusion-rect-20mm.opd`. The pinned
Match Shape scenario uses the bundled 8 mm closed-profile fixture.
The pinned Part7 capture remains host-only.
