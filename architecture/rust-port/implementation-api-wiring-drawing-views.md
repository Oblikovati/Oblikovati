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
