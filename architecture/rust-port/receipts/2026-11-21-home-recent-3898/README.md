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

A [native X11 test](x11-home-grid.log.gz) opened Home from Quick Access,
selected Grid View, and typed `part26`. The [filtered view](x11-home-grid-search.png)
shows Part26 alone. Clearing the query and clicking Search
[restored all four cards](x11-home-grid-restored.png).

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

## Sort headings with the right dock open

The [earlier Size run](compact-size-sort-before.log.gz) showed the descending
order after both clicks. In the compact list, the active heading was clipped
to `Si...`. The list now measures each heading and reserves room for its
direction mark before it assigns width to the cell values.

The [published](published-compact-size-sort.json) and
[pinned](pinned-compact-size-sort.json) cases each passed seven steps. The
[ascending capture](home-compact-size-ascending.png) shows the visible Size
heading and the smallest file at the top. The
[pinned blank-size sort](pinned-compact-blank-size.json) and the
[published column reorder](published-compact-columns.json) also passed.

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

## Full pinned baseline

All 35 pinned scenarios and 357 steps passed on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe. The tested files match `ca8602139`, with API
v0.165.0 and Martensite v0.13.1. The pinned source revision was
`b576445b98cffd686dcc05aa8c017de2a591c8c8` and its feature SHA-256 was
`573d517b49ca58326f70a7c1ebd30509b6f43eb14040efa56e62de36db4028dc`.

The [Cucumber report](full-pinned-home.json) and [JUnit report](full-pinned-junit.xml.gz)
were checked against every declaration. The run produced a final capture and
passed the cleanup hook for each scenario. The
[source report](full-pinned-source.json) confirms that the pinned checkout
stayed clean. The [Go baseline](go-home-reference.log) covers routing and
recording recent-file opens.

All 35 pinned declarations now have R0 evidence. Strict verification still
requires bounded build and runtime inputs under
[#3978](https://github.com/Oblikovati/Oblikovati/issues/3978).
