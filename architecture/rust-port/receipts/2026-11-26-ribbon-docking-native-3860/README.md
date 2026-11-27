# Native Ribbon Docking on Wayland and X11

Twenty-one of the 22 published scenarios passed on each backend, with 223
steps in the passing cases. The floating group list uses the available
panel height. Two downward wheel notches reveal Shape Generator, Work
Features, Pattern, and Create Freeform while scrolling Start 2D Sketch
out of view. The cases also exercised native pointer selection of a
face, switching tabs and document types, Finish Sketch from the left
dock, and the current theme accent for selected groups.
[Wayland scroll capture](floating-scroll-wayland.png)

The Shell scenario stops at its unbound setup. The earlier
[native probe](../2026-11-25-ribbon-docking-current-3860/README.md)
recorded a vertex-tolerance refusal for the same input Part on the prior
kernel pin. Martensite #554 tracks the preview dependency.

The tests ran with `a98ccab593687cbe0c7deba16467d1408783dd59` and an
existing local GUI filesystem test-module visibility edit. Wayland used
nested Weston 13, X11 used Openbox, and both used Xvfb and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source
before and after both runs. Strict build and runtime verification remains
open under #3978.

- Wayland [report](wayland.json), [arguments](wayland-arguments.json), and
  [source check](wayland-source-revision.json)
- X11 [report](x11.json), [arguments](x11-arguments.json), and
  [source check](x11-source-revision.json)
