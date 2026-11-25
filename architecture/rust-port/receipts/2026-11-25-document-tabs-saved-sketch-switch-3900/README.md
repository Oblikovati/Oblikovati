# Switch between the published sketch Parts

The current Document Tabs scenario passed all 14 steps. Both saved Parts
opened in order. The first held only the projected origin, and the
second held four sketch lines. Switching to the second tab updated the
window title and model browser and rendered all four profile edges at
their projected viewport positions. Switching back restored the first
Part without the rectangle and kept the second tab open.

The run used base `779b73f67a1a40bab919fb31fa72c972f7178b91`, the saved-fixture bindings, and existing
local GUI filesystem, opening and saving edits. The native window used
Linux X11 with Xvfb, Openbox and Vulkan Lavapipe. See the
[Cucumber report](cucumber.json) and [input hashes](current-inputs.sha256).
Strict build and runtime provenance remains open under #3978.
