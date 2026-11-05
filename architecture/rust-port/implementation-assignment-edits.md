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

The existing typed API assignment methods are now advertised by the file owner.
Part targets require an active part. Body and face targets additionally require
an injected `PartTopologyAccess`; a file-only host cannot accept raw entity
keys. The desktop supplies this capability from its current named part results.
It resolves the exact feature namespace and checks document, kind, generation,
and the complete key before mutation. Unsupported or failed topology is a named
decline, never an unverified assignment.

The request boundary validates hexadecimal keys and canonicalizes their envelope
and text case. The wire commands reject empty asset IDs, matching Go
`app/material_ops.go`; the internal model setters still support clearing intent.
Part appearance requests require the contract's empty key. All asset dependencies
are prepared before one ordinary history edit. Failure leaves assets, assignments,
and history unchanged. An explicit transaction can abort the complete edit.

Tests drive the API-owned clients, including portable project copies, save/reopen,
undo/redo, transaction abort, scope errors, and missing topology capability. Native
host tests bind actual body/face names, refuse wrong kinds and foreign namespaces,
quarantine failed features, preserve explicit appearance during material edits,
and retain a pending first scene after rejected asset lookup. Reference validation
and rendering share the same result cache; validation cannot consume a display
update that the window has not received.

Full appearance rendering, lost-reference health, remapping across general feature
edits, occurrence publication, and foreground MCPBridge acceptance remain open.
The current native verifier covers the retained sketch-prism capability and does
not claim unsupported feature families. No kernel geometry algorithm or public
wire definition is introduced.
