# Material Browser

The library row's Assign to Selection command uses the selected Part in
the Model Browser. A material change updates the document list, the
ribbon's Material and Appearance selectors, and the rendered Part.
An explicit part appearance retains precedence over the material's
appearance. Rename, Delete, and Add to are displayed as unavailable in
this menu.

All four published Material Browser scenarios passed 30 steps on each
desktop. The assignment scenario selected the published solid Part,
right-clicked its Aluminum 2011 library result, and applied the menu
command. The test verified both copies of the row, the two ribbon
selectors, the catalog color submitted for rendering, and a changed
framebuffer sample inside the solid. The browser stayed open.
Narrow docked search results show the category tree above the table so
the Name, Library, and Category headings remain readable.

All five shared asset browser scenarios also passed 32 steps on each
desktop. The native ribbon asset regressions cover typed assignment,
undo, redo, saving, reopening, and viewport repainting.

The build used base `eaa8a3f28`, the [implementation and test changes](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The [published scenario and fixture hashes](published-inputs.sha256)
matched before and after testing. Strict provenance remains open under #3978.

- [Wayland report](wayland.json), [step arguments](wayland-arguments.json), and [assignment screenshot](wayland-assigned.png)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- Shared browser on [Wayland](shared-wayland.json) and [X11](shared-x11.json)
- Native ribbon tests on [Wayland](ribbon-wayland.txt) and [X11](ribbon-x11.txt)
