# ADR-0038: Own part model-state member recipes

Status: Accepted for the empty-part member lifecycle.

## Context

The application model-browser BDD scenarios require a built-in Primary member,
creation, rename, activation and inactive copies in a part document. The Go
reference owns assembly members in `model/assembly/modelstate.go`. It separates
creation from activation. The Inventor reference gives built-in members immutable
names and stable identities. A browser row cannot own these rules.

## Decision

The model owns an ordered member recipe and an active member identity. Primary
has identity 1. Authored members reserve identities from the document's permanent
recipe allocator, with 1 skipped. The parameter and model-state lookup namespaces
remain separate. Both use the same permanent reservation stream, whose high-water
mark already survives undo and native save. History checks every new authored
member against reservations and checks loaded identities against that mark.

A part native package stores the recipe under `partModelStates`. The section is
separate from the Go assembly representation tuple format. An older part with no
section receives Primary. Other document types receive no part member recipe.
Unknown fields and invalid names, identities or active selections are refused.
The first save writes the explicit member recipe.

The API producer adds rename and copy operations and the Primary flag. Existing
create remains inactive; a UI New command composes create and activate. Copies
remain inactive. No public DTO or wire method name is duplicated in the GPL host.

## Consequences and boundary

This removes the need for UI-owned member names and active check marks. The first
slice covers empty parts. It does not claim per-member feature, parameter or
property overrides, edit scope, property tables, view representation cameras or
annotation scales. Those behaviors require their own owned payload and tests.
No geometry is copied or hidden by this member registry.
