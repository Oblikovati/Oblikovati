# Remove successive staged ribbon commands

The published scenario opens the saved Part with a writable copy of the
Normal ribbon profile. It stages 2D Sketches, 3D Annotations, and 3D PDF,
then selects only the first row. Each Remove click returns the command to
All Commands and selects the next surviving target row. A third click
leaves the target empty and still marked pending. Cancel closes Customize
and leaves the Tools ribbon visible.

The assertions inspect command order and the rendered selection highlight.
The build used `78dfdf004` and the existing local GUI filesystem edits.
Wayland used private nested Weston 13, and X11 used Xvfb with Openbox.
Both used frame-loop input and Vulkan Lavapipe.
[Input hashes](inputs.sha256)

All 19 steps passed on both backends.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Run details](test-results.txt)
