# Resize the main window from its perimeter

All eight published examples passed on each backend, with 80 steps on
Wayland and 80 on X11. The Wayland title-bar move regression also passed
all six steps.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Native resize trace](test-results.txt)

The eight published examples exercise four edges and four corners of
the saved Part window. The test checks the pointer cursor, growth on the
requested axes, the fixed opposite boundary, and the position and size
of any untouched axis. It also checks the ribbon, viewport, and
application controls inside the resized drawable.

The Wayland run uses real pointer input through Weston 13 nested on private
Xvfb with Openbox. Weston's scene graph supplies global application
bounds and the cursor surface. The private Xcursor fixture gives each
resize icon a distinct buffer width so the test can identify the displayed
cursor. The protocol trace records the native resize edge for each drag.
X11 uses Xvfb, Openbox, tint2, and XFixes cursor comparisons. Both runs
use Vulkan Lavapipe.

At baseline `1ddd45080`, each Wayland example opened the Part and then
searched for an X11 Oblikovati window. The verification build used the
backend-aware bindings and the existing local filesystem edits.
[Baseline report](baseline.json) · [Input hashes](inputs.sha256)
