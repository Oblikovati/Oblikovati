# Atomic portable assignment edits — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019. The source behavior is `app/material_ops.go`:
`AssignMaterial`, `AssignAppearance`, `embedMaterial`, and `embedAppearance`.

The document model now prepares a complete assignment candidate from immutable
inputs. Physical targets are part or body; visual targets are part, body, or face.
Entity targets retain the existing occurrence-aware reference value. These values
express authored intent; they do not prove that a body or face exists.

Selection uses document, project, then builtin scope, as the existing resolver
does. A physical assignment copies a non-builtin material and its selected
appearance into the document. A builtin material can still refer to a project
appearance, which must be copied. A visual assignment copies only the appearance
and cannot change physical intent. Every property group remains API-owned.

Missing requested assets or appearance dependencies refuse the candidate before
adoption. The previous document stays unchanged. Builtin definitions remain
references. Empty IDs clear the selected intent without requiring its old asset
to exist, as the model assignment setters specify. Clearing does not delete
embedded definitions; other references can still need them. A material without
an appearance ID needs no visual copy and keeps the resolver's neutral behavior.

Model tests cover property preservation, all target kinds, dependency failure,
builtin references, document scope precedence, occurrence isolation, and clearing.
Integration tests use the ordinary document command owner and named memory
filesystem. They verify one undo step, redo, transaction abort, dirty state,
save/reopen, and typed queries after reopening without any project library.
Run `cargo test -p oblikovati-model --test asset_assignment_edits` and
`cargo test -p oblikovati-automation --test queries assignment_history`, or
`make gate` for the full tree.

The API assignment methods remain unadvertised. Body and face verification must
use current exact topology and persistent names before host commands can accept
these targets. The current retained prism results have construction lineage,
but the file host has no general persistent topology-key resolver. Adding raw key
writes here would bypass that requirement. Rendering, lost-reference health,
remapping across feature edits, and foreground MCPBridge acceptance remain open.
No new kernel algorithm or public wire definition is introduced.
