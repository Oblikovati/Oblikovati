<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Document tab order regression evidence

The native frame-loop tests exercise overflow scrolling, deliberate document and
Home drags, closing and opening documents, and closing an original window after
moving its extra window. The first overflow test failed before the fix: after one
left scroll the visible IDs were `[10, 11, 12, 13, 9]`, instead of
`[9, 10, 11, 12, 13]`. The tests use actual rendered tab positions as well as
owner state; the captures show the resulting strip.

The owner keeps a surviving extra window's original ID, so its order and preview
remain attached to that window. A separate regression checks that hiding and
restoring an open document keeps its dragged position. Another checks that
closing an inactive window keeps the active window and sketch.

The focused captures and four passing native tests use source commit
`421c19c9c`, Linux Xvfb and Mesa Lavapipe. Build provenance remains open under
issue #3978; the frame-loop test is not an OS-input test.

The full feature ran on source commit `421c19c9c`, using pinned scenario snapshot
`05317bbb2d0809a532b29b7ff6f726a5722b865f`. All 38 scenarios and 334 steps
passed. The complete JSON and JUnit reports, source revision, compressed runner
log, and overflow capture are in `full-feature/`. The other feature captures
are not duplicated here.

For the OS-input check, MCPBridge created 14 parts in the standalone desktop.
XTest clicked the left arrow at `(1689, 266)` and moved the pointer away. The
MCPBridge capture shows LivePart11, LivePart12, and LivePart13 in order, with
LivePart14 still active. The final document list confirms the active document.
The application and Xvfb were closed after capture. This is a focused live
check, not a replacement for the full feature run.

The desktop executable SHA-256 was
`2b81706e939e779358ef04797700ca5dc92acfe9d5fdd496d36e6518ede6cd3c`;
the bridge SHA-256 was
`0131e32dcde42491bbfb908973ca8129823947131c597e6ffb9a6a594a999f90`.
The OS run script uses temporary paths; the XTest and MCP probe sources are in
the earlier `2026-11-15-live-tab-overflow` receipt.
