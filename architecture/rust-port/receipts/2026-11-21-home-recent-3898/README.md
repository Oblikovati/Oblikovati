# Home Recent Documents layout

The desktop regression [reproduced](red.log.gz) a Location label that
extended 454 pixels past the Date Modified column. The fix at
`5485d7072` fits the displayed text into each column and leaves the
stored names and paths available to Recent actions.

The [passing regression](regression-final.log.gz) uses a native Part
with a long name and long location. It checks that they end before the
next columns and that hovering shows the full location. The
[list capture](home-recent-long-cells.png) shows readable dates. The
[hover capture](home-recent-location-tooltip.png) shows the full folder.

The [original startup run](startup-final.log.gz) failed after reaching the
Get Started ribbon. Its [capture](home-without-a-document.png) shows the
blank document area and the Model browser's empty state.

## Home on first launch

The [published startup scenario](published-home-first-launch.json) passed
all six steps and its cleanup hook. The [capture](home-first-launch.png)
shows Home beside the Model browser and both asset tabs in the right dock.
A [native X11 click on New Part](home-first-launch-new-part.log.gz) opened
the Part workspace. A saved profile still controls which panels appear.

The [Home navigation scenario](published-home-navigation.json) passed all
eight steps with the asset dock open. Its setup closes the files used to
seed Recent, leaving the Part with the active sketch as an open tab.

The [floating Appearance Browser](published-appearance-docking.json)
and [Material Browser from Quick Access](published-quick-access-material-browser.json)
scenarios exercised docking, resizing, and reopening with explicit starting
states. Both passed.

## Home grid with the right dock open

The [capture](home-wrapped-grid.png) shows all four fixture cards with the
fourth card on the next row. The published
[view switch](published-grid-switch.json), [sort](published-grid-sort.json),
and [selection](published-grid-selection.json) scenarios passed all 28
steps and their cleanup hooks.

## Recent row actions

At the desktop test width, the fixed column offsets put the row's
actions control past the window edge. The list now reserves room for
the control before it sets the widths used by headers and rows. The
[actions scenario](row-menu.json) and [row hover scenario](row-hover.json)
passed all 17 steps after the change. The [actions capture](home-recent-row-actions.png)
shows the popup beside the last column.

The [checked-row actions](row-checked.json) and
[column reorder](column-reorder.json) scenarios also passed all 15 steps.
The [pinned blank-size sort](pinned-blank-size.json) passed its eight steps
with an entry whose saved file had been removed. The current source omits
that scenario.

## Pinned Home baseline

Five scenarios passed all 40 steps and their window cleanup hooks on
`bcabd0c01` under Xvfb, Openbox, and Vulkan Lavapipe. The ledger binds these
R0 entries to the pinned feature hash. The reports record the
[row menu](pinned-row-menu-bcabd0c01.json),
[checked-row menu](pinned-row-checked-bcabd0c01.json),
[row hover](pinned-row-hover-bcabd0c01.json),
[column order](pinned-column-reorder-bcabd0c01.json), and
[file properties](pinned-file-properties-bcabd0c01.json).

## Open with options

The test now checks the Front view that the native Part already saved.
The [pinned](pinned-part-open-options.json) and
[published](published-part-open-options.json) scenarios each passed 14 steps
and their window cleanup hooks. Both selected Front in File Open Options
and returned to Recent with Cancel.

The pinned [assembly](pinned-assembly-open-options.json) and
[drawing](pinned-drawing-open-options.json) cases also passed. They cover
opening-mode selection, control availability, and cancellation without
opening either file.
