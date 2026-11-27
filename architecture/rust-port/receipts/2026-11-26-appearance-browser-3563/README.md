# Appearance Browser

The document and library panes default to independently scrollable 64 px
thumbnail grids. Switching the document pane to List View exposes Name,
Type, and Category columns while the library keeps its previews. Preview
shading uses authored base color, roughness, metalness, anisotropy, and
the OpenPBR fuzz layer. Fuzz is shown as a procedural texture on the
preview. Image texture maps are outside this browser slice.

The library menu assigns an appearance to the selected Part root through
the existing validated assignment command. Assign to Selection is disabled
without an eligible selection. The native scenario verifies that the
assigned project appearance appears in the document, the ribbon shows its
name, the existing steel material remains assigned, and the browser stays
open. Rename, Delete, Where Used, and Add to are visible menu entries with
their actions disabled.

The native tests use the published solid Part, two document preview
fixtures, and a temporary project library with polished and satin brass.
The project library fixture supplies brass because it is absent from the
shipped catalog. The preview check samples the rendered colors and
microfiber stipple on the framebuffer.

All six published Appearance Browser scenarios passed 39 steps per
backend. The Material Browser and shared asset browser reruns passed nine
scenarios and 62 steps per backend. Four native toolbar and assignment
tests also passed per backend. The regression runs used the same production
code before the final Appearance Browser test narrowed its tile assertions.

The build used base `77f34db9e`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The [source and fixture hashes](published-inputs.sha256) matched
before and after testing. Strict provenance remains open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Preview grid](wayland-previews.png) and [assigned brass](wayland-assigned.png)
- Material Browser on [Wayland](material-wayland.json) and [X11](material-x11.json)
- Shared browser on [Wayland](shared-wayland.json) and [X11](shared-x11.json)
- Native toolbar and assignment checks on [Wayland](toolbar-wayland.txt) and [X11](toolbar-x11.txt)
