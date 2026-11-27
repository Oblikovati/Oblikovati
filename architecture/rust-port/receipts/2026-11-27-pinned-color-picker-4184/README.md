# Expanded line color picker

The pinned scenario **Inspect and cancels the custom line color picker**
passed all 13 desktop steps on both Linux backends. Opening the custom
controls showed the palette, preview, and color fields. Canceling both
dialogs left the sketch line color at Default and Apply disabled.

Both runs used the clean MCAD-Scenarios revision
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and Vulkan Lavapipe.
The [X11 run](x11.json) used `9b3267b94`, Openbox, and Xvfb.
The [Wayland run](wayland.json) used headless Weston and the same color
picker code with the marking-menu test setup change `162cead51`.
