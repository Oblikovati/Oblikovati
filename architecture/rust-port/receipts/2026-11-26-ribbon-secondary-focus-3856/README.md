# Give a secondary-clicked command list keyboard focus

A secondary click selects a row and gives its list keyboard focus in
Customize. In the source list, Ctrl+A after the click selects the
visible rows and leaves the target empty. In the staged list, Ctrl+A
selects the staged rows after keyboard focus moves from the source.
The choice of row does not apply or reorder the draft.

The source-list check failed on Wayland and X11 at `559e0ab8d` with the
new test. The secondary click highlighted 2D Sketches, but Ctrl+A
left the other visible rows unselected. The final build includes the
focus change, both native checks, and existing local filesystem edits.
Wayland used headless Weston. X11 used Xvfb and Openbox. Both used
Vulkan Lavapipe. [Test results](test-results.txt)

The published secondary-selection scenario passed the selection, menu,
and empty-target checks on [Wayland](wayland/published.json) and
[X11](x11/published.json). It then expected Cancel to close the clean
dialog. The functional capture documents Close for the clean state
and Cancel for a pending edit. The source folder's `BLOCKERS.md`
records the conflict. [Published input hashes](inputs.sha256)
