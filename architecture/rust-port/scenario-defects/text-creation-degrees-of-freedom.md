# Sketch Text 01 and 13 solver counts

The Sketch Text feature at SHA-256 `0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`
expects a five-degree increase after one Text placement in two cases.
`text-01` records 210 to 215. `text-13` records 28 to 33 and expects
the same result after Redo.

The native tests use generated `.opd` fixtures at both starting counts.
One point Text creates the text entity and its anchor in the sketch.
The point adds two solver freedoms. The tests show 212 and 30 dimensions
needed, respectively. Bold changes the text presentation and Redo
restores the same created entities and glyph.

Check both recordings for the original part, insertion gesture, and any
other geometry or constraints changed in the same action. If the Text
object itself has five freedoms, identify the three variables in addition
to horizontal and vertical anchor position and show how the user can
constrain them. Retain the exact creation and history checks.

[Native fixtures, screenshots, and logs](../receipts/2026-11-21-sketch-text-anchor-freedoms-3907/README.md)
