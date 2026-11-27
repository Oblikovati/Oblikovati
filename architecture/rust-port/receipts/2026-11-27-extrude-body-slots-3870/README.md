# Extrude body slots in Part replay

Part evaluation records each live body by its producing feature and output
slot. Existing single-body producers use slot zero. A grouped producer
namespace resolves a topology key only when exactly one live member accepts
it. Replacing or retiring a slot leaves its siblings in the group. Prefix
capture and restore keep the original slot identifiers.

The grouped replay regression registers two named outputs, checks that an
unqualified or duplicate name is ambiguous, replaces one output, captures
and restores the result, and retires the sibling. The feature tests exercise
existing Part evaluation, replay boundaries, and feature replacement.

The [certified adapter test](../2026-11-27-extrude-profile-batch-3870/README.md)
covers the geometry for disjoint regions. Part recipe authoring and the
desktop selection flow for multiple profiles remain open.

[Replay and compatibility results](tests.txt).
