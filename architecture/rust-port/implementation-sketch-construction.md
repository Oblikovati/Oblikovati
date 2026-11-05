# Owned sketch construction roles — partial R3

This step implements persistent construction roles for owned lines, circles, and
circular arcs. It implements
[issue #3768](https://github.com/Oblikovati/Oblikovati/issues/3768).
Plain points have no construction flag, as in the Go model.

`Sketch2D::set_curve_construction` validates the full selection before mutation.
Duplicate, foreign, unknown, and point IDs decline with the offending selection
and expected owned curve IDs. Changed IDs use authored entity order. Empty and
idempotent selections return no changed IDs. Construction curves still take part
in constraint solving. `profile_curve_ids` excludes them from boundary candidates;
this is candidate selection, not a port of the profile construction engine.

Flags belong to the sketch owner and remain attached to entity identities through
solver candidates. A flag edit changes no coordinates, dimensions, constraints,
entity order, reference keys, or identity reservations. The document command uses
explicit update mode and retains current numerical verification, including failed
evidence. It does not retry the solver for a metadata edit.

`DocumentSession::set_sketch_construction_projected` prepares a response before
history adoption. A response failure or observer veto discards the candidate.
Normal undo, redo, grouped commit, and abort use the same saved roles. An equal
accepted candidate receives the existing history generation tick but creates no
committed undo step and does not make a saved document dirty.

Planar recipe version 3 requires explicit full entity order and construction
metadata. Construction IDs must be unique owned curves in authored order. Restore
rejects malformed metadata before adopting an owner. Existing versions 1 and 2
retain their encoding when no flags are present. Clearing all flags returns to
the applicable version 1 or 2 encoding. Empty version 3 metadata is accepted and
canonicalizes to that earlier encoding. Native YAML packages retain flags across
save, reopen, and copy; a copy retains local IDs and receives new document keys.

The canonical `sketch.entities` response now reports the actual saved flag.
No wire setter was added: API 0.157.0 has no separate construction setter. The
typed session command provides the owned edit boundary for future native actions.
Part and assembly documents use the same command.

Tests cover atomic edits, all owned curve kinds, malformed recipes and packages,
profile candidate exclusion, numerical bit preservation, verification retention,
reference keys, command refusals, response failure, veto, history groups, identity
reservations, and file lifecycle. Pinned Go construction and format-toggle tests
provide independent baseline checks at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

Native selection versus armed construction mode, centerline formatting, included
curves, rendering acceptance, and kernel profile/feature consumers remain required
for the application port.
