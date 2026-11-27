# Selected-face Measure command

The Part context menu exposes Measure for one current selected face. The
action carries its document and face key through dispatch; the owner checks
them against the current selection and resolves the named face before it
calculates analytic area through the existing Part Measure path. A stale, multiple, or mixed selection is not used
as the initial measurement (#4052).

A native X11 test selects a face, opens the menu, starts Measure, checks the
fixture's area, closes the tool, and checks the feature list. The existing
ribbon Measure path also passed its native test. Owner tests cover two faces,
a mixed body and face, and a selection made stale by a parameter edit. These
diagnostics do not
complete the selected-face context-menu scenario: the separate standard menu,
Select Other, Select Tangencies, Properties, and face export remain open.

The Part Measure panel also accepts current named edges through the viewport
picker. Click an edge to measure it and Ctrl-click another to add its length.
The panel resolves each key against the current body and uses Martensite's
analytic curve length. It clears measurements when the model generation
changes. See the
[solid-edge Measure receipt](receipts/2026-11-24-solid-edge-measure-3967/README.md).
