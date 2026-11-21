# Sketch line selection diagnostic

This is a diagnostic, not a verified acceptance receipt. The source was
unversioned, absent from the pinned snapshot, and tagged `needs_capture` and
`needs_fixture`. Its SHA-256 before and after the run was
`f5e24ccd8a393e2ce319b45405b40bc78a1304f3b74b7df99652af5493ef765e`.
Build provenance remains open in #3978.

One filtered scenario and eight steps passed under Xvfb and Vulkan Lavapipe.
The fixture used host commands. Both user clicks used X11 XTEST input. The
test checks the selected line identity and compares visible pixels to the
active theme's normal and selected colors. It checks that an empty-space click
clears selection. The selected-state screenshot was inspected. This run does
not cover the other scenarios in the feature or real hardware performance.
