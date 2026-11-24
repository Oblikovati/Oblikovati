# Cancel component selection for a drawing layout

The first of four populated-layout scenarios passed all 16 steps. The
native English catalog showed the supplied `Populated Layout.odd`.
The [chooser frame](cancel-shared2/drawing-template-previews.png) shows
both sheet thumbnails. The populated thumbnail uses schematic marks for
the saved view and parts-list positions.

Choosing the populated thumbnail and Create opened the component-selection
dialog. Cancel returned to the original active Part with its existing tab.
See the [Cucumber report](cancel-shared2/cucumber.json) and
[final frame](cancel-shared2/final.png).

The test binary used base revision `9307c80e68d928df43a0e687a7356dbfabe3f2ac`
with the preview implementation, bindings, and existing local GUI filesystem,
opening, and saving edits. The run used Linux x86-64, Xvfb, Openbox, and
Vulkan Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source and the isolated copy after the run. Strict build and runtime
provenance remains open under #3978.

The separate native standard drawing creation scenario passed all 14 steps.
