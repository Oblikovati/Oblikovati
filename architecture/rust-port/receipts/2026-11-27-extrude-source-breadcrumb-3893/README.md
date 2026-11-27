# Edit the source sketch from Extrude Properties

Selecting the source sketch in the breadcrumb enters that sketch's
editor and clears the extrusion preview. The compact Properties panel
keeps the feature and sketch visible in the breadcrumb. The active
underline is under the feature before the switch and under the sketch
after it. The scenario samples the rendered underline and checks the
document's active sketch and committed feature count.

The bundled-fixture regression also finishes sketch editing and checks
that Properties closes and the accepted 8 mm extrusion remains.

| Backend | Source sketch | Preview visibility | Finish sketch |
| --- | --- | --- | --- |
| Wayland | [4 passed](wayland-source.xml) | [5 passed](wayland-preview.xml) | [1 passed](wayland-native.txt) |
| X11 | [4 passed](x11-source.xml) | [5 passed](x11-preview.xml) | [1 passed](x11-native.txt) |

[Published input checksums](published-inputs.sha256).
