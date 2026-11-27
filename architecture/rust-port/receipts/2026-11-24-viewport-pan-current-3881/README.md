# Current native viewport pan

The four current scenarios passed individually, with 32 steps. The
published native Part opened with its solid visible in every case. Escape
and a viewport click cancelled Pan. The context menu ended Pan with Done
and switched from Pan to Zoom when requested.

| Scenario | Steps | Native result |
| --- | ---: | --- |
| Activate and cancel viewport pan | 10 | [Cucumber](activate/cucumber.json) |
| Cancel Pan by clicking the viewport | 5 | [Cucumber](click/cucumber.json) |
| End Pan using its context menu | 8 | [Cucumber](menu/cucumber.json) |
| Switch from Pan to Zoom using the contextual menu | 9 | [Cucumber](switch/cucumber.json) |

The build used base revision `f6546c00f619273c4784d0a25d6ee1d74909b4b5`,
the current fixture binding, and existing local GUI filesystem, opening,
and saving edits. The runs used the native window and scripted input on
Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe. The feature and fixture
hashes matched the shared source and isolated copy before and after the
runs. Strict build and runtime provenance remains open under #3978.
See the [combined report](cucumber.json) and [input hashes](current-inputs.sha256).
