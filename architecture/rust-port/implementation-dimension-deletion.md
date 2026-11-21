# Dimensional sketch deletion

Tracking: [#3783](https://github.com/Oblikovati/Oblikovati/issues/3783).

The host consumes the released API's `sketch.deleteDimension` method, request,
result, and typed `client.dimension().delete(&args)` call. It resolves the
checked dimensional index once to a permanent constraint identity. Part and
assembly documents use the same operation. Geometric and structural relations
are outside this dimensional enumeration.

The document operation removes the relation and its owned Model scalar parameter
on an isolated candidate. Owner deletion uses the existing parameter cascade,
which invalidates expression readers instead of leaving stale values. User
parameters and absent parameters cannot be claimed by this operation. An owned
parameter with a failed expression can be removed to recover the sketch.

The selected sketch is evaluated after removal to report current degrees of
freedom. An undefined remaining relation prevents adoption of the entire
candidate. Surviving constraint IDs and order remain unchanged. The operation
does not allocate IDs or release permanent reservations. Normal session
recompute updates other affected readers and feature health.

The normal session boundary prepares the response before adoption. Failed
response preparation and transaction veto restore dimension and parameter
together. Group abort, undo, redo, and persisted reopen use the same recipes.
The wire DOF conversion checks the signed integer range.

Tests exercise all current dimensional forms, protected geometric IDs, survivor
order, source ownership, failed sources, invalidated readers, atomic refusal,
resulting DOF in both owner kinds, malformed requests, history, groups, veto,
response rejection, permanent reservations, and save/reopen. The pinned Go
router deletion tests provide reference evidence. The full local gate covers the
Rust workspace present in this tree. Native viewport and released kernel
acceptance remain open.
