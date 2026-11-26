# Drawing notes

API 0.166.0 `drawingAnnotations.addNote` adds free text to the active
sheet at the requested position in millimetres. A nonzero leader target
adds a line and arrowhead to that point. The sheet can contain notes
without a model reference. Duplicate or omitted names receive a
`NOTE:n` identifier in the current sheet's annotation collection.

`drawingAnnotations.list` lists notes and existing parts lists in sheet
order. `drawingAnnotations.delete` removes a named annotation and
returns the remaining collection. Adding and deleting notes are drawing
history operations. The saved Go-shaped note recipe records the text and
its sheet coordinates. The canvas rebuilds leader geometry on display.
Annotation edits require opening the drawing with updates enabled.

The owner test verifies naming, invalid input, leader geometry, removal,
undo, redo, and save and reopen. The drawing model test verifies that
source-independent notes and source-backed parts lists coexist.

On native Wayland, the MCP Bridge added plain and leader notes to an empty
drawing. In the sheet image, the plain note changed 950 pixels and the
leader note changed 829 more. Deletion, undo, redo, and reopening
reproduced the respective captures pixel for pixel.
