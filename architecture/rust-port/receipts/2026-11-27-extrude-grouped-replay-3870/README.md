# Replay multiple named Extrude solids from a saved Part

One New Solid Extrude can carry multiple directed regions of one sketch in
its authored recipe. The primary retains its existing keys. Each other
output stores a nonzero slot that qualifies its body, face, edge, and vertex
names. All selected regions resolve in the same certified arrangement before
the evaluator publishes their results. Program recipe version 43 records the
group and earlier single-output recipes keep their existing versions.

The rectangle and circle regression checks their volumes and cap areas,
changes the common depth, exercises undo and redo, and encodes and reopens a
document package. It compares the two bodies' exact names across replay and
deletes the circle from a saved checkpoint using its body key. The rectangle
remains live. A second regression checks that a duplicate region or missing
boundary yields zero bodies and that the recipe refuses duplicate and zero
sibling slots.

[Replay test results](tests.txt).
[Body slot receipt](../2026-11-27-extrude-body-slots-3870/README.md).
[Certified arrangement receipt](../2026-11-27-extrude-profile-batch-3870/README.md).

The remaining desktop work is picking and clearing multiple regions, showing
each in the draft preview, and passing their identities through to the same
recipe when the user confirms.
