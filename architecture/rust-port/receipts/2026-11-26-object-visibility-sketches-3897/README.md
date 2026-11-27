# 2D sketch overlays in Object Visibility

The 2D Sketches switch hides the committed sketch geometry and its
dimension overlays. Turning it back on restores each sketch according
to the document's visibility settings. An active edit remains visible.
The focused native check observed the submitted sketch scene and
dimension captions. It verified an unchanged undo list and clean
document after filtering a visible sketch, a stored hidden sketch
staying hidden after both toggles, and an active sketch appearing while
its category is off. The solid remained visible.

All four published checklist scenarios passed 29 steps on each
desktop. Both focused native tests passed on each desktop, and the
sketch-scene module passed its five unit tests.

The build used `7bb562cf3`, the [code and test changes](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston 13 and X11 used Openbox. Both used Xvfb
and Vulkan Lavapipe. [Published inputs](current-inputs.sha256) matched
before and after the runs. Strict provenance remains open under #3978.

- [Wayland report](wayland.json), [step arguments](wayland-arguments.json), and [focused tests](native-wayland.txt)
- [X11 report](x11.json), [step arguments](x11-arguments.json), and [focused tests](native-x11.txt)
