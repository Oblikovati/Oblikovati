# Ceramic document appearances

The Appearance Browser creates and selects a white, glazed Porcelain
appearance. The editor exposes Ceramic and Porcelain types, the shared
base color picker, and three finishes. Matte changes the direct light
and specular highlight in the preview. Apply stores the selected type
and finish alongside the common surface settings in the Part.
Wavy Finish Bumps and Tint have independent enable controls. The Relief
Pattern section reports that no image is selected. The custom bump and
relief image choosers remain open work.

Three Ceramic scenarios passed all 24 steps on Wayland and X11. The
native save test enabled Wavy Finish Bumps, applied the changes, and
decoded the saved Part to verify the type, finish, common surface
parameters, and enabled bump. The Generic creation regression passed
nine steps on each desktop. All ten document package asset tests passed.

The build used base `4908a5bb0`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb with
Vulkan Lavapipe. The runs used a copy of the [published scenario and
fixture inputs](published-inputs.sha256). Strict provenance remains
open under #3978.

- Creation on [Wayland](create-wayland.json) and [X11](create-x11.json)
- Type and finish on [Wayland](finish-wayland.json) and [X11](finish-x11.json), with the [Wayland Matte screenshot](matte-wayland.png)
- Disabled relief on [Wayland](relief-wayland.json) and [X11](relief-x11.json)
- Native saved Part on [Wayland](save-wayland.txt) and [X11](save-x11.txt)
- Generic creation on [Wayland](generic-wayland.json) and [X11](generic-x11.json)
