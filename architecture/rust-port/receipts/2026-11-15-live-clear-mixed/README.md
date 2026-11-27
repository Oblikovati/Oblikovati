<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Live mixed-selection Clear Appearance check

The test ran on Linux Xvfb with Vulkan Lavapipe. The GUI was built from the
worktree based on `a1dffb82b` with the mixed-selection change. Its SHA-256 was
`1a9d8e9aa75fa939edce98f74d1a9a91519dfa5361f202425affb7be96d154b0`.
The MCPBridge library SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
Build provenance is not bounded (#3978), so this is diagnostic evidence.

The GUI opened a copy of the parameter-driven cube. MCPBridge assigned an
appearance override to its body and to each of its six faces as test setup.
XTest selected a face in the viewport and Ctrl-clicked the body in the browser.
`mixed_selected.png` shows the two selected targets and Clear panel. The panel
label in that screenshot had incorrect plurals. After the live run, the label
was corrected and GUI selection validation was consolidated and tightened.
The affected native and GUI tests were rerun.

Cancel preserved the body override and all six face overrides (`before.opd`).
Clear removed the selected body and one face override. Undo restored the
original keys; Redo restored the cleared keys. The saved documents and
`observed.json` record the key sets and file hashes. The owner test separately
checks unrelated assets, exact selected keys, one history entry and reopen.
`input.py.txt` records the XTest driver; its coordinates are specific to the run.

Assembly occurrences remain outside this test. The application and Xvfb server
were closed after the run.
