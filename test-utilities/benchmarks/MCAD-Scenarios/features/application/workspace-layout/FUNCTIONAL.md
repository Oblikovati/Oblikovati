# Workspace layout and panel visibility

The MCAD Software combines a ribbon, model browser, document viewport, navigation controls, document tabs and status area. The View ribbon exposes interface visibility controls. The model browser can be hidden and restored, floated, or docked to either side of the viewport. A docked left browser can be resized by dragging its boundary with the viewport; the neighboring viewport adjusts to the remaining space.

Behavior scenarios are in [workspace-layout.feature](workspace-layout.feature), and schematic visual observations and original icon briefs are in [VISUAL.md](VISUAL.md). Size limits and layout persistence have not been established.

The model browser header menu offers commands to collapse or expand its tree. In the observed part, Collapse All hid nested rows and Expand All revealed them without changing the displayed model.

The browser search field filters the visible tree by entered text. In the observed search, matching work-point names were highlighted and unrelated rows were hidden. Clearing the query restored the unfiltered tree, and closing the empty field hid it.

A query with no matching browser entry shows a no-results message; clearing the query restores the tree.

In the drawing workspace, dragging the browser header into the sheet area floated the browser over the drawing while retaining the sheet and resource tree. Dragging it back to the left edge docked it again and returned the drawing to the remaining workspace.

In the drawing workspace, the browser tab's close control hides the browser and the drawing occupies the freed space. The plus control at the upper left opens a list containing Model Browser; selecting it restores the browser and its drawing tree at the left.

The browser area can contain multiple panel tabs. In the drawing workspace, adding an available log panel through the plus control made that panel active beside the existing Model tab. Selecting Model restored the drawing tree; selecting the log tab returned to the log. Closing the log tab left the Model browser visible.

## Drawing browser tree controls

With a drawing active and browser branches expanded, the browser header menu offered Expand All, Collapse All and Help. Collapse All hid nested entries beneath the drawing resources and sheet while leaving both parent rows and the drawing sheet visible. Expand All exposed nested resource and sheet entries, with the sheet still displayed. This observation does not establish persistence or effects for other document types.

## Drawing browser search

Opening the browser search and entering `Border` filtered the expanded drawing tree to matching border entries under their resource and sheet parent branches. Matching text was highlighted and the drawing remained visible. A query with no matching entry replaced the tree with a no-results message. Clearing the query restored the unfiltered tree; using the close control again on the empty field hid search. Matching rules beyond this observation were not established.

## Drawing browser selection

With the sheet branch expanded, selecting its border entry highlighted that browser row. Selecting the title block entry moved the active row highlight to the title block. The drawing sheet remained visible during both selections. No resulting editing operation or graphical selection highlight was established.

## Search in a floating drawing browser

The drawing browser search remained accessible after floating the panel over the sheet. An unmatched query showed a no-results message; clearing the query restored the tree, and closing the empty search field left the browser floating. The browser was then returned to the left dock. This sequence does not establish matching behavior in the floating panel.

## Move the drawing browser between side docks

Dragging the drawing browser header from the left to the right edge docked it on the right. Its expanded resources and sheet tree remained visible, and the drawing sheet occupied the viewport on its left. Dragging the header back to the left edge restored the left dock with the sheet to its right. The attempted right-dock resize did not establish a width change.

## Toggle one drawing browser branch

Selecting the expansion marker beside an expanded sheet branch hid its child entries without collapsing the expanded drawing resources branch or changing the visible sheet. Selecting the marker again showed the sheet's child entries. The observed nested title block branch also appeared expanded again; persistence beyond this immediate toggle was not tested.

## Search a collapsed drawing resource branch

With drawing resources collapsed, searching for `Border` displayed a matching resource beneath its resource and border parent rows, alongside a matching sheet entry. Clearing the query restored the unfiltered tree with drawing resources collapsed, as it had been before the search. This is an immediate search-and-clear observation; persistence across documents or sessions was not tested.
