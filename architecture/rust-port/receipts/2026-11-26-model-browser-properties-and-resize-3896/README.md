# Model Browser properties and resize on Linux

Three published scenarios passed 35 steps on each backend.

- At `7600af5f8`, the expanded color picker kept Add to Custom Colors
  visible above its scrollbar. Cancel preserved the default line color.
  [Wayland](color-wayland.json) · [X11](color-x11.json)
- At `c95d20a6a`, the sketch line scale field accepted focus only when
  Scale was checked. The value remained 1.00 after clearing the checkbox.
  [Wayland](scale-wayland.json) · [X11](scale-x11.json)
- At `8c2182515`, widening the Model dock narrowed the viewport even when
  the Assets dock moved to share the available space. Moving the divider
  back widened the viewport. The browser and shaded block remained visible.
  [Wayland](resize-wayland.json) · [X11](resize-x11.json)

The native resize test returned the divider to its original position on
both desktops. The color picker unit test kept Cancel visible at two
framebuffer densities. [Focused checks](focused-test-results.txt)

The builds included the existing local GUI filesystem test-module visibility
edit. Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb
and Vulkan Lavapipe. The [input hashes](current-inputs.sha256) matched the
shared source before and after the runs. Strict verification remains open
under #3978.
