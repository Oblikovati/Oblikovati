# Drawing view collection

API 0.166.0 `drawingViews.list` and `drawingViews.delete` use the active
sheet of the active drawing. List returns the ordered authored view
definitions with live visible and hidden curve counts when a source and
projection backend are available. With no projection, the definitions
remain available with zero curves. Deferred drawings show counts from
the saved display geometry.

Delete removes the named base view in one undoable drawing edit and
returns the remaining views. A missing name leaves history untouched.
Drawings opened with deferred updates must be reopened with updates
enabled before deleting a view. The current drawing recipe supports
base views.

The typed regression checks list against the Add Base View reply,
deletion order, an unknown view, undo and redo, and the empty result.
A native Wayland run through the MCP Bridge created Front and Top from
the same saved box, then deleted Front. The drawing crop changed by
1,432 pixels, and undo and redo reproduced both view states exactly.

## Base-view rotation

`drawingViews.rotate` sets a named base view's absolute angle in degrees
on the active sheet. It returns the sheet's view list and records one
undo step. Positive values rotate counterclockwise about the view centre.
`rotationDeg` is stored with the base-view recipe and defaults to zero
for older drawings. Drawings opened with deferred updates must be
reopened with updates enabled before editing views.

The app's projection adapter applies the sheet-space transform after
hidden-line projection. The source orientation, centre, scale, and
visible and hidden classifications stay the same.

The native Wayland MCP run used a 40 × 20 mm rectangular part and
rotated its Top view by 90°. The 2,531 changed pixels were confined to
the Top view. Undo reproduced the original drawing crop pixel for
pixel, and redo and reopen reproduced the rotated crop. The owner test
also checks each projected point against the independent quarter-turn
formula and verifies that its visibility flag stays the same.

## View captions

`drawingViews.setLabel` edits the name, scale, and overall caption
visibility, custom text, and sheet position of a named base view.
Omitted fields retain their values. An empty text override restores the
default caption, and both coordinates must be supplied to move it.
The zero coordinate pair places the caption under the projected view.
The operation is undoable and returns the updated active-sheet view
list. Label settings are optional fields in the drawing recipe.

The typed owner regression covers partial edits, unknown names, undo
and redo, save and reopen, and restoration of default text and position.
On native Wayland the MCP Bridge edited the Top view caption in a
two-view drawing. Undo, redo, and reopening each reproduced the
corresponding sheet capture exactly. Hiding the edited caption changed
580 pixels confined to the caption's 85 × 11 pixel area.
