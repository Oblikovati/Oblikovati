# Ribbon height and compaction

All seven steps of the published sizing scenario passed on Wayland and
X11. In a 1920-pixel restored window, the tab strip and command area kept
the same position and height across 3D Model, Sketch, Tools, Manage,
View, and Annotate. The listed groups displayed their expanded captions,
and the command row had no horizontal scrollbar.

After resizing to 700 pixels, the Modify group rendered a caption,
an icon, and a dropdown marker. The command row kept the same height.
Nine ribbon width calculations also passed, including an exact-fit case.

The test builds used application revision `9a55e493c` and existing local
GUI filesystem edits. Wayland ran on nested Weston 13 and X11 used
Openbox. Both used Xvfb and Vulkan Lavapipe. Strict build and runtime
provenance remains open under #3978.

- [Wayland report](wayland.json) and [X11 report](x11.json)
- [Published input hashes](inputs.sha256)
