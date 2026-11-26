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

## Custom tables

`drawingAnnotations.addCustomTable` places caller-supplied headers and
rows on the active sheet without a model reference. The specified sheet
point is the top-left corner. The grid uses 36 mm columns and 8 mm rows.
Empty cells fill short rows in the presentation. Cells beyond the header
count remain in the saved recipe and are clipped from the presentation.
The existing annotation routes list and delete tables.

The owner test checks missing header validation, blank and uneven rows,
grid coordinates, deletion, undo, and the authored cells after reopening.
The native Wayland MCP run used two columns and three data rows. Adding
the table changed 4,921 sheet pixels. Undo, redo, deletion, and reopen
matched their expected sheet captures pixel for pixel.

## Revision annotations

`drawingAnnotations.addRevisionTable` places revision, date, and
description rows in a table at the given top-left sheet point.
`drawingAnnotations.addRevisionTag` places an upward triangle with the
revision text at the given sheet point. Each is saved with the active
sheet and can be listed, deleted, and undone independently.

The owner test checks required inputs, the table and tag geometry,
separate undo and redo, and the authored rows and tag after reopening.
The native Wayland MCP run placed a two-row table and a B tag on a
drawing with no referenced model. The table changed 7,149 sheet pixels,
and the tag changed another 454. Undo, redo, reopen, and tag deletion
matched their expected captures pixel for pixel.

## Revision clouds

`drawingAnnotations.addRevisionCloud` places an outward scalloped
outline around a sheet rectangle, with an optional tag above it.
The active sheet stores the rectangle and tag. The API returns the
nominal segment count. At small display scales the canvas draws
subpixel scallops as straight edges.

The owner test checks invalid extents, the outward outline, sheet
clipping, and save and reopen. A native Wayland MCP run added a
60 × 40 mm cloud with a B tag and verified 204 nominal segments.
The cloud changed 2,560 sheet pixels. Undo, redo, reopening, and
deletion matched their respective captures pixel for pixel.
