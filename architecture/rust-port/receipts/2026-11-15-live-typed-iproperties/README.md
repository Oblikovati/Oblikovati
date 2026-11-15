<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Live typed iProperties check

On Linux Xvfb with Vulkan Lavapipe, the GUI was built from the task worktree
based on `d1c6f6391`, with the typed-property change in this commit. Its
SHA-256 was `ffd264f4037c1a38dfe42580f7bfe92c8b7e4180ad2601d2e204e16d516fa149`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The build inputs were not bounded (#3978); this is diagnostic evidence. After
the live run, button dispatch and hex chunk iteration were refactored without
changing their behavior.

XTest created a part, enabled iProperties, chose Integer, and entered `-12`
for property `count`. MCPBridge reported its type and value in `created.json`.
Undo removed the property; Redo restored it with the same type and value. The
GUI saved and reopened `properties.opd`; `reopened.json` reports the same
integer. The MCPBridge screenshots show the editor before and after reopen.
`input.py.txt` records the XTest driver; coordinates are specific to this run.

The focused native-window suite separately covers integer, double, boolean and
bytes creation, integer editing, invalid number input, undo/redo, document
switching and save/reopen. Parameter-owned properties are checked by the GUI
and owner unit tests. The application and Xvfb server were closed after the
live run.
