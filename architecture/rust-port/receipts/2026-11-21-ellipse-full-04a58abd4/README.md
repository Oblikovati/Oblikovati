# Full Ellipse run

The unfiltered native sketch suite ran the published Ellipse feature on
application commit `04a58abd4` with Xvfb and Vulkan Lavapipe. It passed 18 of
19 scenarios and 192 of 193 steps. The .feature file, its supporting files,
and all seven native Part fixtures matched the input manifest before and after.

The failed step belongs to `ellipse-05`. After Redo restores the rotated
ellipse, picking the recorded point leaves Measure waiting for a selection.
The selection depends on the native conic extrema fix in
[Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).

The remaining scenarios cover placement, cancel, history, curvature display
and settings, and driving dimensions. The curvature controls used X11 mouse
events. The published scenario directory had no Git revision.

[Rendered curvature comb](ellipse-curvature-visible.png)
