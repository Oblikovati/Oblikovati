# Side-docked 3D Model panel inventory

The published inventory scenario passed all eight steps on Wayland and
X11. Show Panels listed all 19 groups in order, and its default check
states matched the published scenario. A separate native test on both
backends enabled Measure in the direct selector and verified the Measure
and Explore groups in the ribbon column.

The full Wayland visibility suite passed 53 of 55 executions, covering
626 steps in passing cases. The two remaining cases opened Shell
properties and stopped when the preview was absent. The earlier
[Shell probe](../2026-11-25-ribbon-customization-shell-3856/README.md)
records the kernel refusal tracked in Martensite #554.

The runs used the 3D Model changes in `0d4120d94` and existing local GUI
filesystem edits. The published input hashes matched before and after
the runs. Wayland used nested Weston 13, and X11 used Openbox. Both ran
on Xvfb with Vulkan Lavapipe. Strict build and runtime provenance
remains open under #3978.

- [Wayland inventory](wayland-inventory.json) and [X11 inventory](x11-inventory.json)
- [Full Wayland report](full-wayland.json) and [arguments](full-wayland-arguments.json)
- [Published input hashes](inputs.sha256)
