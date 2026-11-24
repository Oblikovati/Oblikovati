# Tools Options expansion

The full Tools ribbon opens additional Options commands below the panel
title. The expansion uses the shared pin control and follows the ribbon
display cycle. Clicking outside an unpinned expansion dismisses it.
In compact panel-button mode, the Options pop-up shows the main and
additional commands together. Clicking its lower title dismisses it.

Reset UI Layout uses the existing ribbon reset confirmation. The reset
restores ribbon placement, appearance, and command customization. Selecting
No closes the confirmation and keeps the current ribbon layout. iLogic
Configuration appears disabled while its host integration is pending.

The [native receipt](receipts/2026-11-24-ribbon-panel-expansion-3856/README.md)
records nine focused published cases and the remaining binding gaps.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

In sketch edit, the Create title's Ribbon Appearance submenu retains Escape.
Clicking outside it dismisses the menu and preserves the sketch. The
[Ribbon Display receipt](receipts/2026-11-24-ribbon-display-full-3856/README.md)
also covers the separate Escape dismissal from the Part Modify title.

Clicking Home, Customize, the Options title, or the ribbon display menu
while the direct panel selector is open dismisses the selector and activates
the clicked control. The display menu can also hand the click back to the
selector. Returning from Home restores the active Part's Tools ribbon and
its panel visibility. The [native handoff receipt](receipts/2026-11-24-tools-selector-handoff-3856/README.md)
covers these transitions.

A right-click on an inactive ribbon tab opens the active ribbon's context
menu. Show Panels uses the active tab's panel inventory, and the inactive
tab remains unselected through the mouse release.

Clearing Modify through the direct selector leaves an unpinned expansion
as a compact title strip. Checking Modify again restores its commands.
Clicking the strip dismisses both the strip and the selector. The selector
button also closes its own checklist on a second click.
