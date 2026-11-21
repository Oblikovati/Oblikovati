# Desktop Fillet radius edit diagnostic

The native window test creates a Fillet from a selected edge, selects its
browser row and opens Edit Feature from the selection toolbar. It changes the
radius by keyboard, checks a private preview and unchanged authored value,
cancels, then repeats and commits. It checks undo, redo, save, and reopen.
The test uses scripted native-window mouse and keyboard events under Xvfb with
Vulkan Lavapipe. It closes the window after the run.

`preview.png` was inspected. It shows the Fillet panel, edited radius and
preview, with no stale error message. The test log records the passing run.
This is a focused diagnostic, not a full BDD acceptance result or a frame-time
measurement. Native build provenance remains open (#3978).
