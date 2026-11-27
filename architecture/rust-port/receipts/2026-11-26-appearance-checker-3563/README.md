# Generic Image menu and Checker texture

The Generic Image menu lists the published texture choices. Checker opens
a separate Texture Editor with a live 2D sample, two colors, softness,
position, rotation, sample dimensions, and independent repeat settings.
The parent appearance preview uses the Checker with the Image Fade
setting. Apply stores the texture in the document appearance recipe, and
the Asset Browser renders the saved texture in its thumbnail. Image import,
other procedural types, Cutouts and Bump images, and viewport texture
rendering remain open.

Both published scenarios passed all 15 steps on Wayland and X11. The
native persistence check verified the appearance preview, a linked size
edit, and a saved Part whose texture was published only after Apply.
The Transparency regression passed another nine steps on each desktop.
All ten document package asset tests passed.

The build used base `1752c3cdc`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The runs used a copy of the shared [scenario and fixture
inputs](published-inputs.sha256). Strict provenance remains open under #3978.

- Image menu on [Wayland](menu-wayland.json) and [X11](menu-x11.json)
- Checker on [Wayland](checker-wayland.json) and [X11](checker-x11.json) with the [Wayland screenshot](checker-wayland.png)
- Transparency on [Wayland](transparency-wayland.json) and [X11](transparency-x11.json)
- Native Checker save on [Wayland](save-wayland.txt) and [X11](save-x11.txt)
