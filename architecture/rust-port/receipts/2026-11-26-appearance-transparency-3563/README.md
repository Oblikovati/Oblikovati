# Appearance Transparency

Transparency has separate disclosure and enable controls. Amount and
Translucency change the transmitted checkerboard preview. The Refraction
list offers Air, Water, Alcohol, Quartz, Glass, Diamond, and Custom.
The selected preset and the custom index are stored in the Part. The
Image and Image Fade controls are visible and inactive while texture
editing is pending. The preset values follow the
[Generic material reference](https://help.autodesk.com/cloudhelp/2016/ENU/3DSMax/files/GUID-80DD9324-A341-4896-B035-17A6163BFBBB.htm).

The published Transparency scenario passed nine steps per desktop. It
verified the initial unchecked state, the controls, a framebuffer change
in the preview, and every preset. The direct native test selected Diamond,
verified the displayed value, entered a Custom index of 1.700, and confirmed
the document retained that value after Apply. The Reflectivity regression
passed 11 steps per desktop. All ten package asset tests passed, including
a round trip with transparency and a custom index.

The build used base `79950dafc`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The runs used a local copy of the latest appearance-editor
source and Part fixture. Its [checksums](published-inputs.sha256) matched
the shared files when copied and remained stable through testing. The
shared file gained another appearance type during the rerun. The two
verified scenarios and the relevant functional text matched the copy.
Strict provenance remains open under #3978.

- [Wayland report](wayland.json), [step arguments](wayland-arguments.json), and [preview screenshot](transparency-wayland.png)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- Reflectivity regression on [Wayland](reflectivity-wayland.json) and [X11](reflectivity-x11.json)
- Native custom index check on [Wayland](native-wayland.txt) and [X11](native-x11.txt)
