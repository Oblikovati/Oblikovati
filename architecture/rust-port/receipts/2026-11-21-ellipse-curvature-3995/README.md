# Ellipse curvature in the native sketch editor

The focused runs use the published Ellipse feature and the native Part
`rotated-measured.opd`. The feature, supporting files, and Part files matched
the input manifest before and after the runs. The published source had no Git
revision.

The curvature menu and settings interactions use X11 mouse events under Xvfb
with Vulkan Lavapipe. The retained capture shows the green comb in the viewport
while Display Curvature is on. The scenario also checks that turning it off
removes the green pixels and leaves the ellipse visible.

The settings scenarios use the rendered controls. They cover both slider
previews, Cancel, Apply, OK, and opening the settings window again. The lower
slider scenario also checks sketch geometry and history after the preview.
These are focused working-tree reports. The feature inventory uses an
unfiltered run to count the full feature.

[Rendered curvature comb](ellipse-curvature-visible.png)
