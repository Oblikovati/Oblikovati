# File Properties Escape

With a saved part selected in Open, the first Escape closes File Properties and
leaves the selected filename and preview visible in Open. The second Escape
closes Open. The native regression reproduced the original failure on Wayland,
where the first Escape also closed Open. The corrected regression and the
existing title-close check passed on Wayland and X11.

The published properties scenario passed all 15 steps on each backend. It
inspected the General, Summary, Project, and Thumbnail pages, then used Cancel
and checked that the same file and preview were still selected in Open.
The published Design View scenario passed all 12 steps on each backend. Its
first Escape dismissed the dropdown and kept Open displayed. The focused GUI
tests for Document Opening and the shared folder browser passed all 29 cases.

The broad run against the unfiltered shared fixtures stopped at file selection
in multiple scenarios because the runner expected the target to be in the
visible directory rows. The focused runs used hash-checked copies of the
[current published inputs](current-inputs.sha256) in a smaller fixture tree.

The build used base `86fb638690cc2ab85ef405fa485f304909d3c3c8` plus this change
and existing local GUI filesystem and test edits. Wayland used headless Weston
with Pixman. X11 used Xvfb with Openbox. Both used Vulkan Lavapipe and frame-loop
input. Restoring a docked File Properties window remains open under #4073.
