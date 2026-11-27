# Page through visible available commands

With focus in the available-command list, Page Down and Page Up first
select the bottom and top visible rows without scrolling. Pressing
either key from its corresponding edge advances the selection by a page
and brings the destination to that edge.

The native regression opens the published Part and measures the available
rows in its actual dialog. From 2D Sketches it checks that Page Down
highlights the visible bottom row and keeps the first page on screen.
Page Up restores the separator highlight. The target stays empty and
Apply stays disabled. The focused GUI unit test also checks paging past
both visible edges and stopping at the ends of the catalog.

The published scenario remains open. Its [baseline](published-baseline.json) stopped at the
empty-target setup wording and its expected bottom row is Adjust
Orientation, which is absent from the current catalog.

The build used `48107e354` with the paging changes and existing local
filesystem edits. Wayland used headless Weston. X11 used Xvfb and
Openbox. Both used Vulkan Lavapipe. [Test results](test-results.txt)
and [published input hashes](inputs.sha256)
