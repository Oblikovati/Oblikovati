# Hole properties at the minimum window size

At 640 × 480, the old Hole panel clipped its trailing Depth caption and
left empty space below its controls. The panel now uses the Studio
property label column, outlined numeric fields, and compact footer.
Through all appears in the empty Depth field. The native X11 regression
opens Hole with Alt, M, H, O, shrinks the window, and verifies the complete
Depth row and viewport bounds. Clicking Cancel closes the panel and
preserves the feature count. [Before](before.png) · [After](after.png)

All four published Hole keyboard scenarios passed with the native Part,
covering 57 steps. Two Hole draft tests and the GUI key-tip test also
passed. [Two-letter tip](case-0/cucumber.json) ·
[Unmatched letter](case-1/cucumber.json) ·
[Minimized ribbon](case-2/cucumber.json) ·
[Left-docked ribbon](case-3/cucumber.json) ·
[Scenario and fixture hashes](current-inputs.sha256)

Verification used Linux X11, Xvfb, and Vulkan Lavapipe. The build used
`2911c9cbd` with the property layout change and pending local
filesystem and Ribbon Customization edits.
