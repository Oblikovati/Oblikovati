# Geometric constraint deletion

Tracking: [#3781](https://github.com/Oblikovati/Oblikovati/issues/3781).
This step implements deletion. Constraint creation remains open in the same issue.

The released API already defines `DeleteConstraintArgs`, `OKResult`, the method
constant, and the typed `client.constrain().delete(&args)` call. The Rust host
consumes those definitions. No new public contract is required.

The host resolves the selected geometric row once to a stable constraint ID.
The model removes that owned ID on isolated geometry, then replaces the sketch
geometry through the existing document operation. Dimensions remain protected.
Structural origin, arc, and projection equations are not authored rows and cannot
be selected through this API. Other authored relations retain their IDs and
insertion order. Geometric indices shift after deletion.

The normal session command performs automatic recompute from current sources.
Deletion can remove a faulty relation from an unhealthy sketch; it does not
require a successful source solve before the edit. The reply is encoded before
adoption. A rejected reply, invalid ID, or transaction veto cannot publish the
deleted candidate. Commit and abort notifications retain the existing generation
contract. Constraint reservation marks remain outside history, so deletion and
undo never release an ID.

Tests exercise all 18 geometric row forms in part and assembly documents, invalid
wire indices and payloads, protected dimensions, structural origin sources,
failed dimension health, response rejection, commit veto, group abort, caught
model errors, permanent reservations, undo/redo, and save/reopen. The old test
that deleted through a recipe rewrite now uses the model deletion operation.

These tests establish the headless command and model behavior on Linux. Native
application rendering and released kernel acceptance remain open.
