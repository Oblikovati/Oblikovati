# Retain the selected 3D Model tab in tabs-only mode

After switching from Tools to 3D Model, clicking the viewport dismissed the
temporary command area. The rendered accent underline stayed under 3D Model,
and the display menu still showed Minimize to Tabs checked. The assertion
reads the accent from the current GUI style and checks the captured framebuffer
under the 3D Model, Tools, and View labels.

All 13 steps passed on Wayland and X11. The build was based on `0ae208217`
with the updated test and existing local GUI filesystem edits. Wayland ran on
nested Weston 13 and X11 ran on Xvfb with Openbox. Both used Vulkan Lavapipe
and frame-loop input.

[Wayland report](wayland.json) · [X11 report](x11.json) · [Selected tab](3d-model-tab-selected.png) · [Input hashes](inputs.sha256)
