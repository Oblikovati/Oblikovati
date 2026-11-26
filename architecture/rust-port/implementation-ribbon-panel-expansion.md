# Tools Options expansion

The [full pinned benchmark](receipts/2026-11-24-ribbon-panel-expansion-pinned-3856/README.md)
passed 57 of 67 scenarios. The ten incomplete cases stopped at an
undefined step. All 57 passing declarations have R0 baseline records.

A focused rerun of the Modify pin case confirmed a
[source conflict](scenario-defects/temporary-modify-pin.md). The pinned
assertion expects the temporary ribbon to stay open after an outside click.
The published case passes with both the temporary ribbon and Modify closed.

The full Tools ribbon opens additional Options commands below the panel
title. The expansion uses the shared pin control and follows the ribbon
display cycle. Clicking outside an unpinned expansion dismisses it.
In compact panel-button mode, the Options pop-up shows the main and
additional commands together. Clicking its lower title dismisses it.

Reset UI Layout uses the existing ribbon reset confirmation. The reset
restores ribbon placement, appearance, and command customization. Selecting
No closes the confirmation and keeps the current ribbon layout. iLogic
Configuration appears disabled while its host integration is pending.

The Highlight New drop-down opens over the Options additional commands.
Escape dismisses the drop-down and leaves the expansion open. Clicking the
selected Tools tab then closes the expansion and keeps the Part visible.
The drop-down reports that no release highlights are available until the
application provides release-highlight data. See the
[published and pinned runs](receipts/2026-11-25-highlight-new-3856/README.md).

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

A right-click on a face outside the open selector dismisses the
selector and passes the point to the native Part selection path.
That gesture shows the face mini toolbar. A later right-click with
the selector closed opens the face context menu.

Hovering over the closed direct selector shows how to add or remove
ribbon panels. Pressing F1 while the hint is visible opens built-in help.
The [native help receipt](receipts/2026-11-24-selector-help-3856/README.md)
covers tooltip dismissal, F1, and switching between ribbon menus.

The Part Tools checklist shows the built-in panels and leaves User Commands
visible on the ribbon. To hide or restore User Commands, use Show Panels
from a ribbon panel's context menu. The published scenario also verifies
that Customize opens while the checklist is shown and removing the last
custom command removes its panel.
[Part Tools native checks](receipts/2026-11-26-part-tools-panel-selector-3856/README.md)

Primitives is hidden by default on 3D Model and can be shown from the
panel selector. The Box, Cylinder, Sphere and Torus controls report their
unavailability when hovered. Space toggles the panel while keeping the
selector open and the same row focused.
[Native keyboard toggle](receipts/2026-11-26-ribbon-primitives-space-3856/README.md)

Showing Primitives moves Modify and its open additional command area together
to the right. Hiding Primitives restores both positions. The selector stays open
after each click.
[Native reflow scenario](receipts/2026-11-26-ribbon-primitives-reflow-3856/README.md)

With 3D Model active, the Show Panels submenu opened from inactive View
or Tools shows Primitives unchecked and Modify checked. The 3D Model
command strip stays visible while the menu is open.
[Native inactive-tab examples](receipts/2026-11-26-ribbon-inactive-tab-panel-menu-3856/README.md)

The display control closes the panel selector and changes the ribbon
display on a single click. The temporary 3D Model command area exposes
the same panel choices. Restoring the full ribbon closes the selector.
[Native minimization scenario](receipts/2026-11-26-ribbon-selector-minimize-3856/README.md)

Clicking the viewport dismisses the temporary commands and keeps 3D Model
underlined in tabs-only display. The selected minimization policy stays checked.
[Native tabs-only selection](receipts/2026-11-26-ribbon-tabs-only-selection-3856/README.md)

The direct checklist wraps keyboard focus from the first row to the last
with Up and from the last row to the first with Down. The
[native regression](receipts/2026-11-24-selector-keyboard-wrap-3856/README.md)
checks the checkbox states after navigation. The published wrap scenario names the
3D Print panel, which is still absent from that catalog.
