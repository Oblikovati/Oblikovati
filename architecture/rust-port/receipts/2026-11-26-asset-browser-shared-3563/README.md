# Shared Asset Browser

The Material Browser and Appearance Browser use the same two-region layout.
The upper region shows document assets and the lower region browses the
built-in and active project libraries. Search filters both regions and
groups library matches by source and category. The display controls apply
independently to each region. This slice uses catalog color swatches for
thumbnail presentation.

All five [shared scenarios](current-inputs.sha256) passed 32 steps per
desktop. The fixture opens the published solid Part and seeds an unused
document copy of red oak and an assigned built-in steel material. The
native steps activate the Material Browser dock tab, check both document
rows, search for aluminum, clear the search, inspect the display choices,
and close the browser while keeping the Part visible. The quick-access
scenario also passed 11 steps per desktop. Four native assignment and
undo tests passed per desktop.

The build used base `cf9ffadc9`, the [implementation and test changes](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and Vulkan
Lavapipe. The shared source and fixture hashes matched before and after the
runs. Strict provenance remains open under #3978.

- [Wayland browser](wayland-browser.png), [scenario report](wayland.json), and [step arguments](wayland-arguments.json)
- [X11 scenario report](x11.json) and [step arguments](x11-arguments.json)
- [Wayland quick-access report](toolbar-wayland.json) and [step arguments](toolbar-wayland-arguments.json)
- [X11 quick-access report](toolbar-x11.json) and [step arguments](toolbar-x11-arguments.json)
- Native assignment tests on [Wayland](native-wayland.txt) and [X11](native-x11.txt)
