# Authored scalar ownership — partial R3

Tracking: [#3746](https://github.com/Oblikovati/Oblikovati/issues/3746), a
prerequisite for [#3745](https://github.com/Oblikovati/Oblikovati/issues/3745).
The existing decisions remain [ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md),
[ADR-0005](../decisions/ADR-0005-identity-and-references.md), and
[ADR-0006](../decisions/ADR-0006-parameters-and-constraints.md).
This implements their stored identity and expression requirements without
replacing an accepted decision or adding a second expression engine.

## Stored ownership

An authored feature stores scalar bindings by stable semantic role and permanent
document parameter ID. The parameter collection retains the expression, units,
health, and dependency edges. The binding stores no evaluated value, callback,
or second copy of the expression. Roles are lowercase ASCII identifiers with
optional digits and hyphens; they start with a letter. Display names are separate.

A binding requires a positive parameter ID and an exact definition read. Roles
and parameter IDs are unique within one feature. The ordered program refuses a
parameter claimed by two features, but permits another feature to read it.
Definition replacement preserves bindings and refuses an edit that removes an
owned read. Rename does not change ownership. Authored equality includes bindings.

Nonempty bindings require mixed datum program version 7. Empty bindings are
omitted, so existing definitions keep their earlier minimum versions and ordinary
recipe layout. Restore applies the same checks as insertion before adopting the
program. This is application recipe metadata, not a new public API DTO.

## Evidence and remaining work

Model tests cover invalid roles and IDs, duplicate roles and owners, external
readers, replacement refusal, rename, exact recipe equality, version refusal,
and malformed stored bindings. The persistence test saves and reopens a plane
whose owned model parameter is `width / 2`, verifies the stored formula and
driver identity, then edits `width` and checks the current plane origin.

The program alone does not prove that a bound ID exists in the document parameter
collection or has model-parameter kind. The
[document ownership checks](implementation-document-scalars.md) now enforce that
boundary and atomic deletion. [Offset authoring](implementation-offset-expressions.md)
now uses that boundary for canonical create/redefine methods and host group/veto
tests. Stored ownership metadata alone does not prove completion of the application port.

Validation is local Linux only. This work changes no kernel checkout, renderer,
Windows/macOS workflow, release pin, or publication claim.
