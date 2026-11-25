# Origin-plane picker projection and compositing

The picker projects a 1.25× expansion of the visible model bounds onto
each origin plane, with a fixed fallback for an empty part. Label glyphs
follow the projected surface, avoid mirroring on back faces, and fade as
the plane approaches edge-on. The projected label footprint is also its
click target. The browser offers each origin plane even when its
viewport label is hidden.

Three GUI projection tests and eight head checks passed. The native
checks used X11 with Openbox and Vulkan Lavapipe. They checked the
Front-view labels, selected a face-on label and an edge-on browser row,
verified the surrounding controls, and confirmed the plane fill reaches
the transparent margin of the ViewCube. Both Extrude resume checks
passed with the shared picker.

[Oblique picker](origin-plane-picker.png) · [Front picker](front-origin-picker.png) ·
[Transparent cube margin](origin-through-viewcube-margin.png)
