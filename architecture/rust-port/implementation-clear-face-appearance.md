# Clear a selected face appearance

Clear Appearance accepts one named face selected in the viewport. The owner
records the exact persistent key and document generation at the pick. On
confirmation it rejects a changed selection or generation and resolves the face
again. A missing or ambiguous name cannot authorize an edit. Clearing removes
only that face override in one history edit; other face and body overrides and
materials stay in place.

The dedicated Linux test target used Xvfb and Lavapipe. Owner tests check a
forged key, stale generation, one history edit, undo, redo, and save/reopen.
The native frame-loop test selects a face after opening Clear and checks Cancel,
Clear and Undo. It injects events into the frame loop; it is not an OS-level
input test or BDD acceptance evidence.

Multiple faces, mixed face and body selections, and assembly occurrence
appearances remain open in #3962. No direct Clear Appearance implementation was found in the Go tree.
