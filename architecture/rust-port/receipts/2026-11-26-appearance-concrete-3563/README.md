# Concrete appearance editing

Concrete can be created from the browser or the editor. The sealant and
finish change the preview and the saved surface parameters. Finish Bumps
has its own Type list. Weathering is enabled by default and offers
Automatic and Custom Image. Tint has its own enable control. The image
chooser embeds sampled stamped finish or weathering maps with the Part.

Both Concrete scenarios passed all 22 steps on Wayland and X11. The
native save check created Concrete from the editor, applied Epoxy and
Polished, imported a PNG weathering image, and deleted the source file.
The Generic copy matched the Concrete framebuffer preview. Decoding
the saved Part confirmed that both appearances retained the map and
that the Generic copy kept its own identity. Ceramic conversion and
chooser cancellation passed another 21 steps per backend.

The build used base `92e32cfed`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The runs used a copy of the [published scenario and
fixture inputs](published-inputs.sha256). Strict provenance remains
open under #3978.

- Sealant on [Wayland](sealant-wayland.json) and [X11](sealant-x11.json), with an [Epoxy preview screenshot](epoxy-wayland.png)
- Finish and Weathering on [Wayland](finish-wayland.json) and [X11](finish-x11.json)
- Native save and Generic conversion on [Wayland](save-wayland.txt) and [X11](save-x11.txt)
- Ceramic conversion on [Wayland](duplicate-wayland.json) and [X11](duplicate-x11.json)
- Ceramic image cancellation on [Wayland](ceramic-wayland.json) and [X11](ceramic-x11.json)
