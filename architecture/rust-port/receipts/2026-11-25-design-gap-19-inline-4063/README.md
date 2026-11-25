# Normal ribbon command sizing

The inline Normal ribbon uses a 64 px command grid inside the 88 px
command band. Full-height commands use 64 px cells. The two-row stack
uses 30 px cells separated by 4 px, and dense Sketch groups use 20 px
cells separated by 2 px.

The native X11 regression measured the fixed overall band and checked
the 34 px row pitch in 3D Model and Tools, and the 22 px row pitch in
Sketch. The captures show each layout at 1440 × 900 and 100% UI scale.
Material and Appearance share a label column and align their dropdowns.
The native checks also exercised Application Options, the full-width
Modify flyout, and the larger icon scale.

[3D Model](model-command-band.png) · [Tools](tools-command-band.png) ·
[Sketch](sketch-command-band.png)
