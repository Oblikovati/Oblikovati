<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Live multi-face Clear Appearance check

On Linux Xvfb with Vulkan Lavapipe, the desktop was built from the task
worktree based on `f48c2c064`, with the multi-face change in this commit. The
binary SHA-256 was
`d270ffdb6e8256becd77019b40c4d971a1d4bdfc22c9c16220e1672de91c5c5a`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
Build provenance is not bounded (#3978), so this is diagnostic evidence.

A copy of the earlier parameter-driven cube was opened through the GUI. The
MCPBridge assigned `abs-black` to its six face keys as test setup. XTest then
selected Faces, opened Clear, chose two visible faces with a click and a
Ctrl-click, and pressed Cancel. The MCPBridge screenshot `two_faces.png` shows
both outlines and the selection count. A save after Cancel contained all six
face overrides (`before.opd`).

The same two selected faces were accepted in Clear. Saves after Clear, Undo and
Redo contained four, six and four face overrides respectively. `observed.json`
records the exact key sets and the SHA-256 of each saved document. The four
retained keys are a subset of the original six; Undo restored the original set
and Redo restored the cleared set. `after_redo.png` shows the final window.
The owner tests separately verify the exact selected keys, unrelated assets,
one history entry and reopen. The XTest script is retained as `input.py.txt`;
its paths and screen coordinates are specific to this run.

This does not cover mixed body-and-face selections or assembly occurrences.
The application and Xvfb server were closed after the test.
