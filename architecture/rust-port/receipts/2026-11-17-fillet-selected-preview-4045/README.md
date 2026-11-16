# Selected edge in a Fillet preview

The native-window test starts Fillet, clicks an edge, checks its selected pixel
while the replacement preview is visible, and confirms the feature. It runs
under Xvfb with Vulkan Lavapipe and uses scripted native-window input. The
capture was inspected; it shows the selected input edge over the preview and
the panel identifies one edge. The source edge is a display overlay, not a
new edge on the preview body or a modeling reference derived from pixels.
The test checks hover and retained-selection colors after moving the pointer.
The edge toolbar is hidden while the Fillet panel is active.

An owner test checks that the exact selected edge outline is retained during
the Fillet preview. The Extrude edit test checks that its replacement preview
continues to hide the selected feature outline. The generic live scenario
still needs its fixture and reference capture, and native build provenance is
open (#3978); this is not a verified BDD acceptance receipt.
