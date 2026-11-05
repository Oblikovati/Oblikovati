# Inferred line creation

Tracking: [#3779](https://github.com/Oblikovati/Oblikovati/issues/3779).
The canonical API constructor remains under
[#3775](https://github.com/Oblikovati/Oblikovati/issues/3775).

## Document command

`DocumentSnapshot::create_inferred_sketch_line` creates fresh endpoints and
an owned line in a private geometry copy. It solves existing geometry with the
current parameter values, projected point sources, and supplied solver options
before selecting inference. A converged sketch can have free variables. A
failed source solve returns a named error before identity allocation.

The shared selector proposes endpoint coincidences and one direction relation.
Coincidences are appended before direction. Applied direction evidence carries
the actual constraint identity and its zero-based position in the geometric
constraint enumeration. Dimensional constraints do not enter that enumeration.
The line retains independent construction and centerline roles.

Geometry is adopted once after all allocations and relations succeed. The
entity and per-sketch constraint reservations remain permanent on exhaustion,
reply rejection, commit veto, undo, and abort. Catching an error inside an edit
cannot publish a partly created line.

The session and workspace commands use document preferences and working-unit
snap reach. They prepare the caller's response inside the normal command path,
then recompute and publish the candidate. An open edit group retains its label.
Save, reopen, and save-copy retain constraints, roles, and reservation marks.

## Go reference and intentional changes

The reference is Go commit `aa8297d1b0ac33c024f19e54821536d76e8a447a`,
`model/sketch/inference_apply.go` and `inference_apply_test.go`.
The horizontal, priority selection, endpoint snap, and disabled inference
reference tests pass with the isolated Go oracle workspace.

Rust keeps proposals separate from applied records. When automatic constraint
application is disabled, proposals remain visible, coordinates remain unchanged,
and no applied direction or constraint index is fabricated. The Go audit mode
can report a direction record with `Count()-1` despite adding no constraint;
the Rust command does not preserve that defect. Fresh endpoints never target
each other, including zero-length inputs.

## Validation scope

The model tests cover all four direction families, ordered snap relations,
actual identities, filtered indices, construction and centerline roles, disabled
constraint application, zero-length input, invalid coordinates and policy,
entity and constraint exhaustion, undo and abort, current parameter changes,
failed sources, and nonconverged sources. Generated integration cases cover
16 rotations, three scales, and both endpoint orders.

Session and file owner tests cover command groups, current solver evidence,
working units, reply rejection, commit veto, save, reopen, and save-copy.
The local gate covers every current Rust workspace member, lint, documentation,
SPDX, coverage, and duplication. It does not prove a native viewport or kernel
integration. Linux remains the current target; Windows and macOS are deferred.
