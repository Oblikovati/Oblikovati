# Ribbon Customization Name header

The Name sort control spans the available-command list at its current
width. The native test clicks at 90% of the rendered list width, verifies
the separator and descending Zoom rows, and transfers the same selected
2D Sketches command. With the earlier label-sized button, that click
left the list in ascending order.

The two focused native tests passed on both Linux backends. Four published
scenarios passed all 52 steps on Wayland. They cover selection and transfer,
staged order, category changes and reopen, and resizing with a staged edit.
The Wayland capture shows the Name header aligned with the source list edges.
[Published report](published-wayland.json) · [Wayland capture](name-header-wayland.png)

The build used `10247fd99` with the header change, the native regression
and existing local GUI filesystem edits. Wayland ran on headless Weston,
and X11 ran on Xvfb with Openbox. Both used Vulkan Lavapipe.
[Native results](native-results.txt) · [Published input hashes](inputs.sha256)
