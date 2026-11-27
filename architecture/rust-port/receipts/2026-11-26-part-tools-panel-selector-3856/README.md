# Part Tools checklist with a custom panel

The published scenario passed all 15 steps on Wayland and X11. Confirming
the customization creates User Commands on the Part Tools ribbon. The
direct checklist shows the seven built-in panels. Customize opens while it is
visible, and removing the last custom command removes User Commands.
Before the change, Wayland reached the checklist and reported an extra
User Commands row.

A separate regression hides and restores the Part Tools custom panel
through Show Panels in the ribbon context menu. The left-docked 3D
Model scenario verifies toggling Modify through its direct checklist.

The build used application revision `a261bb76d`, the changes in this
commit, and existing local GUI filesystem edits. Wayland used nested
Weston 13. X11 used Openbox. Both ran under Xvfb and Vulkan Lavapipe
with input through the native window's frame loop. The published inputs
matched before and after the runs. Strict provenance remains open under #3978.

- [Wayland before](wayland-before.json)
- [Wayland after](wayland.json) and [X11 after](x11.json)
- [Published input hashes](inputs.sha256) and [test results](test-results.txt)
