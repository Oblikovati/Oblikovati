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

The [published startup case](startup-final.log.gz) reaches the Get Started
ribbon and then fails because the New Part action is on that ribbon
and Home is not open. The [startup capture](home-without-a-document.png)
also shows the Model browser's empty state and no asset browsers in the
right dock. The scenario folder's `BLOCKERS.md` records the difference
between this fresh profile and the expected starting layout.

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
