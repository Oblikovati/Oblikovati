# Cut through the complete body with Through All

The native test opens the circular sketch on the 8 mm base. It first
enters a 1 mm numeric distance, then selects Through All. The draft
switches to Cut and Flipped and the Output menu disables Join. The test
checks the shaded preview and probes the upper and lower caps at the
axis and outside the circle. The accepted result is one seven-face
solid with an opening through both caps.

[Wayland JUnit](wayland/junit.xml) and [X11 JUnit](x11/junit.xml)
record the published scenario. The bundled pinned revision predates
this scenario. Both runs use the package identified by the
[Extrude Join receipt](../2026-11-27-extrude-join-3870/README.md).
