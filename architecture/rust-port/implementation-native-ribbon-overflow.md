# Native ribbon panel overflow

Issue [#3849](https://github.com/Oblikovati/Oblikovati/issues/3849) tracks the
width-dependent panel behavior from Go `head/ui/ribbon_collapse.go` and
`chrome_ribbon.go`. The fit calculation keeps leading panels expanded and
collapses panels from the right. Each collapsed tile opens that panel below
the ribbon. Expanded panels and flyouts use the same contents and owner actions.

The GUI owner retains measured widths by tab and panel. A panel without a
measurement expands once so its width can be recorded. Changes to measurements
or scrollbar state set `GuiFrameActivity.pending_layout`, so the host schedules
the frames needed to reach a stable layout. Resizing can restore expanded panels.

A horizontal scroll region keeps the tiles reachable when the collapsed row is
wider than the window. A flyout also scrolls when its full panel exceeds the
window width. The band reserves space for its scrollbar. The status line,
document strip, and browser use the resulting band height. The File menu and
quick-access row stay above the scrolling panel region.

The child-window and layout-space calls cross the C++ boundary as scalars,
following the Windows ARM64 ABI repair in #3803. Theme editor launch requests
return to the ribbon owner before it opens the modal, because a panel flyout
has its own window and popup scope.

## Verification

The GUI suite passed 117 tests, including eight fit tests and 32 ribbon pointer
tests. Pointer cases cover 320, 640, and 1024 logical pixels at both display
scales, popup commands, disabled transactions, Escape, context changes, clipped
children, and restoring expanded panels. The 320-pixel cases reach both the last
tile and the last command of a panel wider than the window through scrolling.

Native X11 acceptance at scale 1 also passed. OS mouse and keyboard input created
an XY rectangle through the collapsed Create panel at width 640. Both dimensions
used `size = 10 mm` and the sketch reached DOF zero. The run exited and reentered
the sketch, armed Construction through Format, and cancelled a private circle
preview with the authored entities unchanged. At width 320 it scrolled both the
tile row and the Constrain flyout, then activated Tangent and cancelled it.

After returning to wider windows, the run changed size to 20 mm, undid and redid
the change, saved, closed, and reopened the document. A fresh process reopened
it again at width 640. Both `size` expressions and healthy DOF zero remained.
MCPBridge supplied read-only queries and exact PNG captures, which were inspected.
The accepted desktops closed normally.

Native X11 uses a backing ratio of 1. The two-ratio claim above covers GUI
coordinate conversion. Native 200% content scale, font/icon preferences, and
actual platform backing-ratio 2 acceptance remain open in
[#3850](https://github.com/Oblikovati/Oblikovati/issues/3850).

The full application gate passed across all 13 crates. GUI coverage was 98.32%,
viewport coverage 93.40%, and head coverage 92.76%. Duplication was 1.02%.
Cross-target Clippy passed for the viewport on Windows x64/ARM64 and macOS
x64/ARM64, and for head and GUI on Windows x64 and macOS ARM64. These are
compile checks; platform runtime checks remain separate. The kernel runtime
blocker in [Martensite#388](https://github.com/Oblikovati/Martensite/issues/388)
and the profile/Extrude blocker in
[Martensite#390](https://github.com/Oblikovati/Martensite/issues/390) remain open.
