# Current Document Tabs verification on Wayland and X11

All 39 published scenarios and 343 steps passed on both backends.
Reordering an active drawing kept its sheet and browser visible. Dragging
an inactive Part moved and activated it while keeping the drawing open.
The focused Wayland runs recorded compositor motion and button events for
both reorder cases.

During the first full Wayland run, the compositor placed the detached
drawing over the Part tab. The test now uses an exposed point on the main
title bar to raise the window before clicking a covered tab. The final
Wayland run passed that case, and a focused X11 detachment rerun passed
after the change.

The build used application revision `968993d52`, the test changes, and
existing local GUI filesystem edits. The full X11 run used Xvfb and
Openbox after the reorder changes. The final Wayland run used the desktop
shell in nested Weston 13 with an outer 2048 × 1200 Xvfb screen at scale 1.
Both used Vulkan Lavapipe. The feature, fixture, and support hashes matched
before and after verification. Strict build and runtime provenance remains
open under #3978.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Wayland reorder trace](wayland-reorder-input.txt) · [Input hashes](inputs.sha256) · [Test results](test-results.txt)
