# Optional Generic appearance effects

Cutouts, Self Illumination, Bump, and Tint each have a disclosure and a
separate enable checkbox. Expanding an unchecked section shows disabled
controls, leaves Apply disabled, and preserves the preview and document.
Self Illumination offers a filter color, luminance, and color temperature.
Tint offers a color filter. Editing those colors affects the appearance
preview. Apply retains the authored settings with the Part. Cutouts and
Bump image slots remain inactive pending texture editing.

All four outline examples passed 28 steps per desktop. The direct native
Bump check edited Amount, applied it, disabled the effect, and confirmed
that the last value remained in the document. The Transparency regression
passed nine steps per desktop. All ten package asset tests passed,
including a round trip with optional effect settings.

The build used base `9aa8c86b8`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The runs used a local copy of the shared [scenario and fixture
inputs](published-inputs.sha256). Strict provenance remains open under #3978.

- [Wayland outline report](wayland.json), [step arguments](wayland-arguments.json), and [Self Illumination screenshot](illumination-wayland.png)
- [X11 outline report](x11.json) and [step arguments](x11-arguments.json)
- Transparency on [Wayland](transparency-wayland.json) and [X11](transparency-x11.json)
- Native Bump check on [Wayland](bump-wayland.txt) and [X11](bump-x11.txt)
