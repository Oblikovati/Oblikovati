# Sketch overlap selection diagnostic

This is diagnostic evidence, not verified acceptance. The pinned source is
local snapshot `b576445b98cffd686dcc05aa8c017de2a591c8c8`. The scenario has
`needs_fixture` and `needs_capture` tags. The native build provenance gate
remains open (#3978).

One filtered scenario passed all four steps under Xvfb and Vulkan Lavapipe.
The host creates a line and a point at the same screen location. X11 XTEST
clicks the point, opens Select Other, and selects the line. The checks identify
the selected entity, check the visible point cue and menu alternatives, and
confirm that geometry and undo history stay unchanged. This does not test a
physical mouse or a production GPU.
