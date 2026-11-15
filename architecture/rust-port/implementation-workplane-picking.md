<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Work-plane viewport picking

The part viewport includes healthy, visible work planes in its existing
selection batch. It intersects the camera ray with the evaluated plane frame
and checks the authored display rectangle. It does not read pixels or triangle
meshes for a model reference. A geometry that the overlay cannot display is
not pickable; its display decline is reported by the overlay path.

Plane, face, and body hits use the user's selection filters, category priority,
and camera-derived depth window. Equal-depth plane identities are declined as
ambiguous. A hidden, suppressed, failed, edge-on, behind-camera, or out-of-bounds
plane does not become the selected feature. Control and Shift toggling of
multiple work planes is not supported and reports a selection notice.

Selecting one plane uses the existing browser-feature selection. The existing
sketch and offset commands then consume that same feature identity. This path
currently covers part documents, not assembly occurrences or origin overlays.
Other selection categories remain subject to the coverage limits in #3890.

Native tests use frame-loop mouse input to select an authored plane and start a
sketch or create a second offset plane. Owner tests cover hidden planes, filter
settings, competing hit priority, and ambiguous overlays. These are diagnostic
checks, not verified BDD receipts (#3978).
