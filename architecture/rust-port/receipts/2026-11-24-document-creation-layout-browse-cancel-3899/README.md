# Cancel browsing for a drawing-layout component

The native Browse-cancel scenario passed all 16 steps. Setup opened a
disposable copy of the supplied Part, updated it, added an unsaved parameter,
and confirmed `RequiresUpdate` before the next viewport frame. The nested
file browser opened above component selection. Cancel returned to the
component field with the same document name. The second Cancel left the
original Part active with its existing tab.

The [Cucumber report](browse-cancel-second/cucumber.json) and
[final frame](browse-cancel-second/final.png) came from base revision
`ca26b74aa93277a63680377360abda970e91df78` with the Browse-cancel bindings
and existing local GUI filesystem, opening, and saving edits. The run
used Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the shared source and the
isolated copy after the run. Strict build and runtime provenance remains
open under #3978.

Two of the four populated-layout scenarios now have native evidence.
Confirmation still needs drawing placement and model-backed view and
parts-list results. The current component dialog disables OK and the
drawing projection path rejects assembly sources.
