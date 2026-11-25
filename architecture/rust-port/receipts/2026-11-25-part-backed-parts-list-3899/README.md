# Parts list on a live drawing

The initial probe used the published extrusion Part and a disposable copy
of the populated layout. It retargeted the drawing's existing model
reference to the Part copy. The authored sheet, Front view, and parts list
came from the layout package.

The baseline Wayland run opened the drawing and displayed the Front row.
It failed because the sheet had no PART NUMBER heading.

The maintained regression builds its Part and drawing from repository
test data so it can run without the published scenario checkout. It edits
the referenced Part's Design Tracking properties while the drawing is
active and checks the table's row count, its quantity, and the visible Part
Number and Description. It also checks that the authored table appears
beside the Front view in the drawing browser.

The test build uses `895b02bb1` with this change and existing local
filesystem and sketch-marking test edits. Wayland uses headless Weston
and frame-loop input. X11 uses Xvfb and Openbox. Both use Vulkan Lavapipe.
[Fixture and scenario hashes](inputs.sha256).

The populated-layout confirmation and stale-model warning are still open
under #3899. This check opens the drawing through the File menu.

The maintained regression passed on Wayland and X11. All nine drawing
model tests passed. [Results](test-results.txt) · [Wayland capture](wayland.png)

The published cancellation reruns stopped before cancellation on both
backends. The outer case expected a specific thumbnail pixel color while
the capture showed both thumbnails. The nested case expected a separate
text run for the proposed component, while the rendered input joined
the field caption to the component name. These checks need follow-up.
