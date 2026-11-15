<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Off-screen tab selection diagnostic

On 2026-11-15, the native `manual_scroll_does_not_highlight_an_inactive_document`
test passed on Linux with Xvfb and Mesa software Vulkan. The test opens sixteen
parts and a presentation, then scrolls the active presentation out of view.
It checks every visible part tab against Home's inactive background, scrolls
the presentation back and checks its selected background, then clicks a
part and checks that only it is selected. The application keeps the expected
active document throughout each scroll. All three captures were inspected.

The test failed before the fix: Part5 had the selected background while the
presentation was active and off-screen. The native harness injects input into
the frame loop, not through OS-level keyboard or mouse events. The capture is
diagnostic; build provenance remains open (#3978).
