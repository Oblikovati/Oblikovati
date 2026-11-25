# Native Customize entry from Sketch Create

The [entry case](case-115/cucumber.json) passed eleven steps from the
Create panel title menu. Customize showed Ribbon, Keyboard, and Marking
Menu, with All Commands and an empty Part Sketch target.

The [selection case](case-116/cucumber.json) passed nineteen steps.
Selecting 2D Sketches enabled the Add control and left the target empty
and Apply and Cancel disabled. Closing Customize kept the published
Part and its editable Sketch1 geometry unchanged. This run reused the
open Sketch1 from the fixture step.

The [input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)
matched on Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe. The entry
test used the resize change in `d5d7e84ca`. The selection test also
used the setup and checks in this commit. Both builds included the
existing local GUI filesystem, opening, and saving edits.
