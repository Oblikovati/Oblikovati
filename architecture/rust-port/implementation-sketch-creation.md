# Checked sketch creation — partial R3

Tracking: [#3763](https://github.com/Oblikovati/Oblikovati/issues/3763).
Reference: Go `app/sketch_env.go` and `addin/router/sketch.go` at migration
baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The document owner creates sketches through `create_document_sketch`. The
authored request carries a fixed frame or stable work-plane reference, optional
orientation, name, and the caller's origin-projection preference. Application
preference defaults and document-kind selection belong to the host. The model
does not copy the public wire request or own application options.

Creation reserves a permanent sketch identity, then builds a private candidate.
It resolves the current host and orientation before adoption. A missing host,
invalid reference, unavailable axis, or degenerate orientation declines creation.
The command never substitutes a base frame after an orientation failure.
Validation also runs when origin projection is disabled.

When enabled, origin projection uses the existing current analytic projection
seam. It reserves an owned point ID, seeds that point in the current frame, and
stores an associative origin binding. No copied fix constraint is added. The
normal session solve removes grounded coordinates from the numerical unknowns
and verifies the new anchor. Later host or orientation edits resolve the origin
again through the same binding path.

The session and workspace command capabilities use one creation implementation.
Creation and reply preparation run inside the normal history boundary. A failed
reply discards the candidate. Failed commands and aborted groups retain permanent
identity marks; a subsequent command cannot reuse either sketch or point IDs.
Undo and redo restore authored definitions and rebuild current verification.

Tests cover enabled and disabled projection, a translated and reversed oriented
frame, a current owned host and later host motion, missing and zero hosts,
degenerate and non-finite orientations, both exhausted namespaces, typed causes,
repeated creation, rejected replies, aborted groups, undo/redo, and native
save/reopen. Workspace tests use the existing named file and observer fakes.

The wire `sketch.create` route, application option storage and default-on
preference, assembly policy, interactive edit environment, face/edge reference
catalogue, and native head still require ports. This command supplies their
shared document mutation boundary. It does not establish native rendering or
kernel acceptance.
