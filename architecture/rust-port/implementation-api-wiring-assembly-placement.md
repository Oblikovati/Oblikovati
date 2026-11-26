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
