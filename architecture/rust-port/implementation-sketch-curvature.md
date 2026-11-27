# Sketch curvature display

In an active planar sketch, right-click an ellipse and select **Display
Curvature**. The same command hides the comb. **Setup Curvature Display** opens
a dockable settings window while the comb is visible. The upper slider changes
the number of teeth. The lower slider changes their length. The sliders show
relative positions without numeric values.

Moving a slider previews the change. **Apply** keeps the current settings and
leaves the window open. **OK** keeps the current settings and closes the window.
**Cancel**, Escape, and the window close button restore the last applied
settings. The application keeps the display settings for each curve in the
current view session. The Part file and sketch history store authored geometry.

The comb uses second derivatives of the analytic Martensite curve. Teeth stay
normal to the curve and preserve relative curvature magnitudes. The sketch
renderer draws the comb as a transient overlay outside model bounds. Analytic
picking uses the authored curve.

The [native receipt](receipts/2026-11-21-ellipse-curvature-3995/README.md)
covers display, density, scale, Apply, OK, and Cancel with X11 mouse input.
The [full Ellipse run](receipts/2026-11-21-ellipse-full-04a58abd4/README.md)
passed 18 of 19 scenarios. The rotated Measure pick remains open in
[Martensite #406](https://github.com/Oblikovati/Martensite/issues/406).
