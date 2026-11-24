# Current native viewport orbit

All nine current declarations passed, including both examples of the
opposite-direction outline. The ten native cases executed 94 steps.
They cover free and constrained Orbit, the visible rotation guide,
incremental rotation preview, Reset, Cancel, OK, and all three
opposite-direction pairs.

| Scenario or outline | Cases | Steps | Native result |
| --- | ---: | ---: | --- |
| Open and exit the free-rotation mode | 1 | 13 | [Cucumber](free-ring/cucumber.json) |
| Select and exit constrained orbit | 1 | 11 | [Cucumber](constrained-ring/cucumber.json) |
| Preview and reset an incremental view rotation | 1 | 14 | [Cucumber](reset-face/cucumber.json) |
| Accept an incremental view rotation | 1 | 10 | [Cucumber](accept-face/cucumber.json) |
| Reverse an incremental rotation with the adjacent control | 1 | 9 | [Cucumber](reverse/cucumber.json) |
| Reverse a rotation using the remaining adjacent pairs | 2 | 18 | [Cucumber](pairs/cucumber.json) |
| Exit free orbit with Escape | 1 | 6 | [Cucumber](escape/cucumber.json) |
| Leave free orbit using the Pan control | 1 | 6 | [Cucumber](pan/cucumber.json) |
| Exit orbit with the context menu OK action | 1 | 7 | [Cucumber](ok/cucumber.json) |

The first two cases use the same runs as the earlier
[guide receipt](../2026-11-24-viewport-orbit-current-guide-3872/README.md).
The remaining builds used base revision `e9978b07e1866a01d3913b334432fd71b5a5ebd2`, the
updated fixture bindings, and existing local GUI filesystem, opening,
and saving edits. The runs used the native window and scripted input
on Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe. The feature and
fixture hashes matched the shared source and isolated copy before
and after the runs. Strict build and runtime provenance remains open
under #3978. See the [combined report](cucumber.json) and
[input hashes](current-inputs.sha256).
