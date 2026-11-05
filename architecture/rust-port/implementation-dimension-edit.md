# Dimension target, role, and limit edits

Tracking: [#3785](https://github.com/Oblikovati/Oblikovati/issues/3785).
Go reference: `model/sketch/dimension.go`, `dimension_test.go`, and the dimension
router in `addin/router/sketch.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The part and assembly hosts implement the released API's
`sketch.driveDimension` contract through `client.dimension().edit(&args)`.
The canonical request, method constant, and result remain API-owned. The host
resolves an ordered dimension index once to a permanent constraint ID.

## Target behavior

A nonempty expression drives the feature-owned Model scalar parameter. The
existing expression service resolves named parameters, units, and categories;
bare values use the document display units. A drive freezes the evaluated value
as a constant, as in Go. It does not retain a new live formula. An invalid,
incompatible, non-finite, missing, or non-owned target leaves the entire candidate
unchanged. A failed owned parameter can be repaired by a valid drive.

`setLimits` enables finite ordered lower/upper bounds. Length bounds arrive in
centimetres and convert to working units; angular bounds remain radians. Setting
bounds does not clamp an existing value. The next drive clamps the new value,
including when the bounds were restored from storage. Rust refuses reversed or
non-finite bounds instead of accepting an inconsistent interval.

`setDriven` changes whether the relation is a reference dimension. Reference
relations remain authored and measurable but contribute no solver equation or
target parameter dependency. Their queries report measured geometry, name,
expression, annotation, and `driven: true`; an unused failed target does not block
measurement. Returning to driving restores the equation and source dependency.
A failed target then produces failed sketch health. No solve success is fabricated.
Role or limit edits alone do not overwrite the target. Omitted flags and an empty
expression preserve the corresponding fields.

## History and storage

Model edits use an isolated candidate. Session and workspace wrappers prepare the
reply before adoption and follow normal recompute, history, group, and veto rules.
Repeated recipe bits add no undo step. The normal owner generation can still
advance when a command succeeds without changing the recipe.

Planar recipe version 6 stores explicit owned reference IDs and drive limits in
authored dimension order. It retains entity order, construction and centerline
roles, and annotation metadata. Versions 1-5 remain readable without fabricated
settings. Restore rejects missing metadata, duplicate or foreign IDs, invalid
bounds, and wrong order. Deleting a dimension removes its settings. Signed-zero
bounds remain distinct for recipe identity. The outer document schema remains 3.

Tests cover all 24 dimensional variants in reference mode, absent and failed
sources, drive and clamp behavior, parameter readers, display units, category
errors, three working scales, angular units, malformed recipes, both host kinds,
both query hosts, groups, vetoes, rejected replies, undo/redo, save/reopen, and copy.
Pinned Go tests verify reference dimensions, limits, and the radius drive path.
Run `make gate` for the complete Linux application workspace checks.

Dimension creation, text layout, native rendering, and feature/kernel evaluation
remain separate port work. This change does not complete R3 or the application port.
