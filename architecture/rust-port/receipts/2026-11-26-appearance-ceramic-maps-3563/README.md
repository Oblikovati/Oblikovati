# Ceramic bump and relief images

Enabling Finish Bumps makes Type and Amount editable. Custom opens the
shared file browser, with PNG, JPEG, and BMP choices. Cancel keeps the
current type and map. Relief Pattern uses the same chooser when enabled.
The editor uses 32×32 grayscale height samples for preview shading and
embeds the samples in the applied Part.

The Custom cancellation scenario passed all 14 steps on Wayland and
X11. The disabled Relief Pattern regression passed seven steps on each
backend, and Generic Keep Editing passed nine steps per backend. The
native Wavy save check passed again on both. A separate native test imported two different PNGs, verified
the preview, deleted the source images, applied the draft, and decoded
the saved Part with both embedded maps. All ten document package asset
tests passed.

The build used base `269690a99`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The runs used a copy of the [published scenario and
fixture inputs](published-inputs.sha256). Strict provenance remains
open under #3978.

- Custom cancellation on [Wayland](cancel-wayland.json) and [X11](cancel-x11.json), with the [Wayland screenshot after Cancel](wavy-after-cancel-wayland.png)
- Disabled Relief Pattern on [Wayland](relief-wayland.json) and [X11](relief-x11.json)
- Native import and save on [Wayland](save-wayland.txt) and [X11](save-x11.txt)
- Generic Keep Editing on [Wayland](keep-wayland.json) and [X11](keep-x11.json)
- Native Wavy save on [Wayland](wavy-wayland.txt) and [X11](wavy-x11.txt)
