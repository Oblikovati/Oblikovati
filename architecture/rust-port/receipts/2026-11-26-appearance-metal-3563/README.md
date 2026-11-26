# Metal appearance editing

Metal can be created from either the browser or the editor. The authored
alloy sets the base color. The finish changes the highlight and
roughness, with anisotropy for Brushed. The implementation uses the
four captured Brass finishes for all eight alloys. Relief Pattern,
Cutouts, and Tint have separate controls. Imported relief and cutout
maps affect the appearance preview and are embedded in the applied Part.

The published Metal scenario passed all 14 steps on Wayland and X11.
A native test saved Brushed Brass and an imported PNG cutout after
deleting the source image. The saved Part retained the alloy, finish,
metalness, anisotropy, and the embedded map. Cube preview, Concrete
sealant, and Ceramic-to-Generic duplication passed 27 regression steps
per backend. All ten document package asset tests passed.

The build used base `c538d394b`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The runs used a copy of the [published scenario and
fixture inputs](published-inputs.sha256). Strict provenance remains
open under #3978.

- Metal on [Wayland](metal-wayland.json) and [X11](metal-x11.json), with a [Brass preview screenshot](brass-wayland.png)
- Cube preview on [Wayland](cube-wayland.json) and [X11](cube-x11.json)
- Concrete sealant on [Wayland](concrete-wayland.json) and [X11](concrete-x11.json)
- Generic duplication on [Wayland](copy-wayland.json) and [X11](copy-x11.json)
- Saved Metal on [Wayland](save-wayland.txt) and [X11](save-x11.txt)
