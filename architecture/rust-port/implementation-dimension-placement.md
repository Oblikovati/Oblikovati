# Dimension annotation placement

Tracking: [#3784](https://github.com/Oblikovati/Oblikovati/issues/3784).

The host consumes the released API's `sketch.moveDimension` request, method,
result, and typed `client.dimension().move_(&args)` call. Part and assembly
owners share the same command. A checked dimensional index resolves once to a
permanent constraint ID. Text positions are finite sketch-plane points in
working length units; the API converts to and from centimetres.

Placement is authored metadata, separate from constraint equations. A move
retains numerical verification, geometry, source expressions, measured values,
and DOF. The explicit session update mode does not solve the sketch. Failed
sources do not prevent annotation-only editing. The session prepares the reply
before adoption and uses normal history, veto, and group boundaries. Identical
placement bits are a history no-op; signed-zero changes remain distinct.

Planar recipe version 5 stores explicit dimensional placements in authored
constraint order. It includes prior entity, construction, and axis metadata.
Version 1-4 recipes remain supported and receive no fabricated placement.
Unknown IDs, geometric relation IDs, duplicate or reordered placement IDs,
nonfinite positions, missing version-5 metadata, and metadata in legacy versions
are refused. Unannotated recipes retain their prior version. The outer document
schema remains 3. Deleting a dimension also deletes its placement.

Tests cover model ownership and finite values, recipe round-trip and corruption,
legacy migration, bit comparisons, deletion, retained verification, no-op history,
reply rejection, veto, group abort, failed sources, both owner kinds, scales,
malformed API requests, unit overflow, undo/redo, save/reopen, and copy. Go router
annotation tests provide reference evidence. The local gate covers the Rust
workspace present in this tree. Native label rendering, dimension authoring,
reference mode, and released kernel acceptance remain open.
