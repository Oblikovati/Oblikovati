# Resume Extrude after finishing an existing profile

The published scenario opens Sketch1 from the saved rectangle fixture,
starts Extrude from the 3D Model ribbon while the sketch is being
edited, and finishes the sketch. Extrude keeps the source sketch
active until Finish Sketch. Properties opens with one selected profile
and shows a shaded 10 mm preview. No feature has been committed.

The empty-part entry scenario continues to request an origin plane
and open Sketch1 for drawing. The bundled native regression also
draws a new rectangle and resumes Extrude.

| Backend | Empty part entry | Resume existing profile | Resume newly drawn profile |
| --- | --- | --- | --- |
| Wayland | [5 passed](wayland-entry.xml) | [7 passed](wayland-resume.xml) | [1 passed](wayland-native.txt) |
| X11 | [5 passed](x11-entry.xml) | [7 passed](x11-resume.xml) | [1 passed](x11-native.txt) |

[Published input checksums](published-inputs.sha256).
