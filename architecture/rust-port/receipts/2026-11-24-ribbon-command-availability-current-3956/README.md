# Current native ribbon command availability

All five current scenarios passed individually, with 39 steps. The
published revolve-profile Part has one sketch and no solid. The
tests checked command availability outside Sketch edit, on return
to the Sketch tab, and while editing Sketch1. The Assembly checks
left the saved document idle. The Drawing checks covered Place Views,
Annotate, and visibility of the drawing sheet.

| Scenario | Steps | Native result |
| --- | ---: | --- |
| Inspect Sketch commands outside sketch edit | 7 | [Cucumber](outside-final/cucumber.json) |
| Revisit Sketch from another ribbon tab | 9 | [Cucumber](revisit/cucumber.json) |
| Inspect Sketch ribbon while editing an existing sketch | 13 | [Cucumber](existing/cucumber.json) |
| Inspect component commands in an active assembly | 4 | [Cucumber](assembly/cucumber.json) |
| Inspect view and annotation commands in an active drawing | 6 | [Cucumber](drawing/cucumber.json) |

The build used base revision `4d12def5cad4159460b0befbc2f899827650e10a`, the
updated fixture bindings, and existing local GUI filesystem, opening,
and saving edits. The native window used scripted input on Linux X11,
Xvfb, Openbox, and Vulkan Lavapipe. The feature and fixture hashes
matched the shared source and isolated copy before and after the runs.
Strict build and runtime provenance remains open under #3978. See the
[combined report](cucumber.json) and [input hashes](current-inputs.sha256).
