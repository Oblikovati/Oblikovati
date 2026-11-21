# Ellipse alias and dimension scenarios with saved Parts

The published scenarios ellipse-12, ellipse-18, ellipse-19, and ellipse-20
passed with cleanup in the desktop window under Xvfb and Vulkan Lavapipe.
The corresponding pinned scenarios also passed. The runner uses scripted
pointer and keyboard input.

The alias cases open the published native circle and midpoint-line fixture.
They use the specified free center. The digit appears in the command alias
status, and both existing entities and sketch history remain intact. The
viewport comparison uses a capture with the sketch layer hidden as its baseline.

The dimension cases open the supplied dimensioned Part. A zero minor radius
shows the failure dialog and leaves the 4 mm expression in place. Finish
Sketch returns to an oblique view with the ellipse and both dimensions visible.
Edit Sketch restores RIGHT view with the same annotations. The test prepares
the oblique view before opening the sketch while keeping the fixture's zoom.

These are focused results. Strict acceptance depends on sealed build and
runtime inputs under [#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
