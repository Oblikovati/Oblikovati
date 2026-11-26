# Assembly placement by definition

The Rust file host handles `assembly.placeByDefinition` and
`assembly.placeByDefinitionBatch` from API 0.166.0. It resolves the source
occurrence in the active assembly and reuses the same authored component
reference held by that occurrence. Each new occurrence owns its own name,
placement, and session ID. Copies of an already placed component are
ungrounded.

The host validates the names and affine matrices before editing. It adopts
the batch and encodes the ordered result in one projected document command.
An empty request returns an empty list. One invalid placement leaves the
whole assembly unchanged.

The typed file-host regression checks rejection of a singular matrix after
a valid placement in the same request, empty input, ordered names and
transforms, stable IDs across redo, and one undo for the entire batch. It
also checks single-copy placement and the shared component reference.

A disposable native Wayland run used the MCP Bridge to save a 40 mm box,
place the first occurrence, and place 15 more in one batch. The reply
returned all 15 new names, IDs, and transforms in order. The live assembly
listed 16 occurrences, then one after undo and 16 after redo. A captured
frame showed all sixteen boxes in a four-by-four grid. The driver set an
explicit camera frame for the capture.

## Occurrence transforms and grounding

The API can reposition, ground, and release an occurrence in the active
assembly by its session ID. It validates the affine transform before the
edit and keeps the occurrence's component binding and ID. A repeated
setting keeps the current history cursor. Each changed setting is one
undoable command. The file-host regression verifies invalid inputs,
undo and redo, and the persisted transform and grounding after reopen.

Contact-set collision blocking depends on the assembly contact solver,
which is still pending for the Rust desktop.

The Wayland MCP run released the first grounded occurrence in the grid,
moved it, undid and redid the transform, then grounded it at the new
position. The transform changed 31,606 pixels in the viewport crop. Undo
restored the original crop pixel for pixel. The frame after redo and
grounding matched the moved crop.

## Occurrence removal

The API deletes an occurrence by its session ID and returns the surviving
tree in assembly order. Unresolved component files appear in this reply
as authored occurrences with no child entries. A missing file can
therefore be repaired or removed by ID.

The remaining copies continue to share their document binding. When the
last copy is removed, the workspace releases its clean hidden component
after the transaction commits. Undo resolves that component again. A
nested transaction leaves the reference available for an outer abort.
The file-host regressions cover these cases, a transaction veto, and
save and reopen after removal.

The Wayland MCP run removed one occurrence from a 16-box grid. The reply
contained all 15 surviving IDs in order. The viewport crop changed by
20,719 pixels. Undo restored the original crop exactly, and redo restored
the removal crop.
