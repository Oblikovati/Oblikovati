# Extrusion direction selector styles

The desktop Extrude dialog starts with four labeled direction icons from the
existing asset set. Its Direction menu offers Icons and Dropdown. Both selectors
use the same four directions. The dialog retains the style for the desktop
session, including after Cancel.

The native run caught Escape dismissing the popup and canceling the draft in the
same frame. Dialog shortcuts now defer to the popup that owned the key.

The pinned checkout remained clean at
`b576445b98cffd686dcc05aa8c017de2a591c8c8`. The feature SHA-256 was
`ccb973b10334aa2f8f31a189392ad8049212caaa2938db03b034bc9f49008821`.
At `1bfda4ede`, `extrusion-03c` stopped at its first scenario-specific step
because there was no matching binding.

All 12 steps of pinned `extrusion-03c` passed under Xvfb and Openbox. The
scenario checks the selected menu row and the rendered icon next to each
direction caption. A separate native test selected Symmetric in the list,
verified the centered preview after switching styles, and reopened the
Dropdown after Cancel. Eight dialog tests, six ribbon interaction tests,
and Clippy passed.

The four pinned direction scenarios passed all 28 steps together. Pinned
`extrusion-04` through `extrusion-06` also passed all 20 steps, covering Cancel
and reopen, invalid-distance recovery, and Undo/Redo with measurement after
acceptance.

See the [Cucumber report](pinned/cucumber.json), the [expanded
Dropdown](pinned/direction-dropdown.png), and the [restored
icons](pinned/direction-icons.png). The [native selector
test](native-selector.log) records the session-style check.
