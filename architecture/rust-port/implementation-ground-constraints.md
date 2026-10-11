# Entity ground constraints — partial R3

Tracking: [#3790](https://github.com/Oblikovati/Oblikovati/issues/3790).
Boundary: [ADR-0015](../decisions/ADR-0015-parameters-and-constraints.md).
Reference: Go revision `aa8297d1b0ac33c024f19e54821536d76e8a447a`,
`constraints_new.go`, `edit_ops.go`, and `constraint_kind.go`.

## Behavior and ownership

The released `ground` constructor now accepts any currently native entity:
point, line, circle, arc, or ellipse. It captures each defining point from the
current solved owner seed through the normal typed geometric command. The curve
identity selects its points; the persisted relation and canonical enumeration
retain their ordered point identities, as the Go `RelatedEntities` method does.
There is one authored constraint identity and one numerical residual block.

Each coordinate residual subtracts its frozen target. The existing single-point
fix shares this coordinate equation; no second numerical solver or entity-specific
ground engine is introduced. Derivatives are exact coordinate unit rows. Ground
does not freeze scalar radius or orientation variables. An ellipse therefore
retains three freedoms after its center is grounded. A circle retains its radius.
An arc grounds its center and both endpoints, while its structural circularity
equation remains in the normal shared constraint system.

Owned distinct points and finite coordinates are required. An empty, duplicate,
foreign, or non-finite anchor set cannot be adopted. Incompatible frozen anchor
sets report their authored constraint identities through the usual diagnostics.
Fixed-point inspection includes the defining points without calling a free radius
or orientation fixed. The former canonical ground constructor decline is removed.

## Storage and acceptance

Planar schema 8 adds frozen ground anchors. Old schemas remain readable and
refuse ground records. Schema 8 carries explicit order, construction, centerline,
dimension placement, reference, limit, and ellipse fields. Frozen coordinate bits
participate in history comparison, including signed zero. The relation survives
save/reopen/copy, and deletion does not reuse its permanent constraint identity.

Tests cover both document owners, every native operand shape, rank, exact
derivatives, three length scales, invalid anchors, conflicts, enumeration,
undo/redo, group abort, commit veto, reopen/copy, and deletion. The strict coordinate
scale sweep sets an explicit numerical tolerance: the current solver scale also
includes angular variables, so the default relative tolerance is not a physical
length tolerance. These tests do not claim a new general tolerance policy.
The pinned Go `TestGroundConstraintFixesEntity` supplies baseline behavior evidence.

Full Linux validation is `make gate`. This work introduces no kernel operation or
native UI runtime. Grounding does not supply a missing minor-radius dimension;
fully constrained ellipse acceptance remains open in #3788.
