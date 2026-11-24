# Current native viewport zoom

All seven current scenarios passed individually, with 74 steps.
The native file dialog opened the published Part for every case. Real
X11 scroll events changed its magnification while keeping its
orientation and the active Part. The navigation bar cases checked
Zoom activation, Zoom All, Zoom Window, Zoom Selected, and cancellation.

| Scenario | Steps | Native result |
| --- | ---: | --- |
| Change magnification with the mouse wheel | 8 | [Cucumber](wheel-final/cucumber.json) |
| Activate and cancel Zoom from the navigation bar | 10 | [Cucumber](activation/cucumber.json) |
| Fit a distant block using Zoom All | 11 | [Cucumber](fit-all/cucumber.json) |
| Zoom into a rectangular window | 15 | [Cucumber](window/cucumber.json) |
| Fit a selected planar face | 12 | [Cucumber](fit-face/cucumber.json) |
| Choose a face after starting Zoom Selected | 10 | [Cucumber](select-after/cucumber.json) |
| Cancel a zoom window after its first corner | 8 | [Cucumber](cancel-window/cucumber.json) |

The build used base revision `216b13e044035d606567b5e2a0081a919216cac1`,
the updated scenario bindings, and existing local GUI filesystem,
opening, and saving edits. The runs used Linux X11, Xvfb, Openbox, and
Vulkan Lavapipe. The feature and fixture hashes matched the shared
source and isolated copy before and after the runs. Strict build and
runtime provenance remains open under #3978. See the
[combined report](cucumber.json) and [input hashes](current-inputs.sha256).
