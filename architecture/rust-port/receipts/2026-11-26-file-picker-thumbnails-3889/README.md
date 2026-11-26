# File picker thumbnail rendering

The published **Switch between details and thumbnail file views** scenario
aborted on Wayland and X11 before this change. The Wayland debugger measured
66,920 vertices in the Open dialog's file-list draw list when Dear ImGui
checked its 16-bit index range.

Off-screen tiles still take up their normal row space. The picker skips
their preview geometry and text. The GUI owner advertises support for
vertex offsets, which the frame snapshot already applies when it remaps
indices. A regression captures 17,000 rectangles in one draw list and
checks the index count and the position of the final rectangle.

Once the crash was fixed, the same scenario showed that the Details column
headings could scroll out of view. The headings now stay above the scrolling
rows. A [follow-up fix](../2026-11-26-file-picker-name-column-3889/README.md)
kept long filenames from overlapping the modification-date column.

Both thumbnail scenarios and the size-sort scenario passed all 29 steps on
each backend. The Save As view-switch scenario passed all 14 steps on Wayland
from a disposable copy of `SaveAsNativeFolder`. The 87 focused GUI tests
passed, including the draw-list regression.

[Wayland thumbnail report](wayland.xml) · [X11 thumbnail report](x11.xml) · [Published input hashes](current-inputs.sha256)

The build used base `6e7d950b9b491712f6a0a0f33c6cdf6337b7bf97`, this fix,
and existing local GUI filesystem and test edits. Wayland used headless
Weston with Pixman. X11 used Xvfb with Openbox. Both used Vulkan Lavapipe and
frame-loop input.
