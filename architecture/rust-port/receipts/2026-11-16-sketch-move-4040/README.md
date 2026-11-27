# Sketch Move preview diagnostic

This is diagnostic evidence, not verified acceptance. The pinned source is
the local snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`, not an upstream
revision. The scenario has `needs_fixture` and `needs_capture` tags, and the
native build provenance gate remains open (#3978).

One filtered scenario passed all nine steps under Xvfb and Vulkan Lavapipe.
The test creates the initial line through the host. It uses X11 XTEST to open
the Move tool, select the line and base point, move the pointer, and press
Escape. The checks cover preview pixels, unchanged sketch geometry and undo
history, and preview removal. The before and after frames were inspected.
This does not test a physical mouse or keyboard.

The filtered Line creation scenario passed nine steps after sharing the
preview-removal step. Ready-state selection passed eight steps with the shared
line fixture. The collapsed Modify-panel test also passed. Other scenarios in
the feature were not run.
