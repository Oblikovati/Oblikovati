# Compact ribbon groups and chrome

The compact renderer assigns a catalog glyph to every ribbon group. It
puts the disclosure beside the caption and uses an inline icon when the
chosen appearance has room for only one row. The published narrow View
case exercised Appearance and Windows, including repeat clicks and
dismissal. The narrow Manage case exercised Insert and restored its
inline commands after maximizing. Both passed on X11 with Openbox.
[Compact View capture](view-compacted.png) · [View scenario](view/cucumber.json) ·
[Manage scenario](manage/cucumber.json)

The Normal tab strip reserves 28 logical pixels and centers the density
and panel controls. The native geometry check verified the 32 px titlebar,
the 28 px tab strip, and the stable 88 px command band. It also read the
16 px brand glyph and its gap to the first Quick Access icon from the
submitted quads. [Titlebar and tab strip](ribbon-chrome.png)

Normal Modify flyout commands draw as full-width 28 px rows with 16 px
icons. The native test verified the row pitch and opened Move Bodies by
clicking near the right edge of its row. The pinning and outside-click
File dismissal scenarios also passed.
[Modify flyout](normal-modify-flyout.png) · [Pin scenario](pin/cucumber.json) ·
[File scenario](file/cucumber.json)
