# Stable Sketch ribbon and idle Light ViewCube

The Normal full ribbon's command region measures 88 logical pixels.
Sketch Create and Constrain fit within three command rows. Their authored
widths are available before the first draw, so the compact layout can show
Finish Sketch immediately at 920 × 680. The viewport's top edge stays
fixed across tab and width changes. The horizontal scrollbar is hidden
from the main band. Horizontal scrolling still reaches the final group
below the minimum width of the compact buttons.

The first-frame GUI regression verifies the same viewport top for 3D Model
and four successive Sketch frames, with Finish Sketch wholly in bounds
each time. The native capture uses Xvfb and Lavapipe and resizes the editing
workspace from 1440 × 900 to 920 × 680 before clicking Finish Sketch.
[Expanded Sketch ribbon](sketch-ribbon-compaction.png) ·
[Compact Sketch ribbon](sketch-ribbon-920x680.png)

In Light mode the visible ViewCube labels retain the slate ink
`#1E2430` at full alpha even as the faces and controls fade. At zero
opacity the label also disappears. The native test samples the rendered
label with the pointer away. The published inactive-opacity scenario
passed all 12 steps and checked that 100% and 50% still have distinct
face opacity and preserve the Part view.
[Idle Light ViewCube](light-viewcube.png) · [Opacity scenario](opacity/cucumber.json)
