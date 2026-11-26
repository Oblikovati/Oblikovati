# Full Model Browser rerun on Wayland

The current published feature passed 96 of 100 scenarios, with 1,305
steps in passing scenarios. The other four stopped at undefined setup
steps for the extrusion-owned angle parameter d1. They are:

- Parameter interface follows the active model state
- Add and apply a second model-state parameter
- Show active member parameter values in the table tree
- Inspect parameter column and cell context menus

The four setup steps require the extrusion taper contract tracked in
Oblikovati #4127 and Martensite #463. Every executed step before those
undefined steps passed. This run covered the fixes documented in the
[Model Browser implementation evidence](../../implementation-model-browser.md).

The build at `a9fdab737` included the existing local GUI filesystem
test-module visibility edit. It ran on nested Weston 13 with Xvfb
and Vulkan Lavapipe. The [published input hashes](current-inputs.sha256)
matched before and after the run. The [binary hash](binary.sha256) was
recorded at launch. Strict build and runtime verification remains
open under #3978.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
