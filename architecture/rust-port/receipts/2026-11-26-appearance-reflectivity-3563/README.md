# Appearance Reflectivity

The Generic editor separates the Reflectivity disclosure from its enable
checkbox. Expanding the section shows disabled Direct and Oblique
sliders and leaves the editor clean. Enabling the effect makes both
controls editable, changes the preview sphere, and enables Apply.
Direct affects the face-on reflection and Oblique affects the rim in
the preview. Apply updates the document asset and its browser thumbnail.
The saved document retains both strengths when the effect is disabled.

The published Reflectivity scenario passed 11 steps on each desktop.
It compared the drawn sphere before and after enabling the effect and
verified that the document asset stayed unchanged before Apply. The
native slider test applied separate strengths, then disabled the effect
and verified that both values remained. The Information regression
passed seven steps on each desktop. All ten package asset tests passed,
including a round trip with distinct Direct and Oblique values.

The build used base `df60ba36b`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The [source and fixture hashes](published-inputs.sha256) matched
before and after testing. Strict provenance remains open under #3978.

- [Wayland report](wayland.json), [step arguments](wayland-arguments.json), and [preview screenshot](reflectivity-wayland.png)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- Information regression on [Wayland](information-wayland.json) and [X11](information-x11.json)
- Native slider check on [Wayland](native-wayland.txt) and [X11](native-x11.txt)
