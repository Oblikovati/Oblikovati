# ADR-0005: Persistent identity and geometric references

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R1/R3/R5/R6/R8.

Boundary: names across kernel operations, recipe replay, and external imports.
Owner: application model; Martensite owns operation-history production and
geometric resolution algorithms. Oblikovati.API owns public reference values.

## Context

Rebuilding a recipe destroys object addresses and can change topology order.
A successful key decode does not prove that the new kernel will resolve it to
the intended entity. Legacy geometric recovery also conflicts with the current
rule that a missing identity must not silently bind to nearby geometry.

## Decision

Use one persistent naming mechanism derived from operation provenance and
semantic feature roles. Assign a stable namespace per feature instance, separate
from its editable display name. Martensite must report generated, modified,
deleted, and surviving entities with their complete parent sets. The model
combines that history with feature and occurrence context; it does not infer
history from tessellation or proximity.

Maintain identity through split, merge, transform, and healing. Preserve keys
for surviving entities and retain all parent relationships on merged entities.
Where one old entity has several descendants, resolution must use an explicit
semantic mapping or return ambiguity. It must not select the first descendant.
Entities without direct history derive identity from named neighbors where
possible, in either topological direction. New authored entities receive stable
recipe IDs, not build counters.

Version every ordering used to distinguish equal-parent results. Define total
orders, bounded recursion, cycle detection, and source-count limits. Verify a
key when writing it: resolve it and confirm the same intended entity. Occurrence
paths use stable placement identity, not array positions or shared pointers.

## Interfaces and invariants

- Read supported legacy key bytes and the versioned little-endian `OKRK`
  envelope. Preserve its context, kind, scheme, and payload semantics. Reject
  malformed or unsupported encodings with specific errors.
- Do not put geometric anchors into identity bytes. A key miss or ambiguity
  is an identity error, not an invitation to try nearest geometry.
- External geometric references remain a distinct tagged selection mode for
  translators that cannot provide Oblikovati keys. Classify this mode once;
  resolve against analytic geometry with documented units/tolerances. Return
  one certified match with visible recovery health, or a lost/ambiguous error.
  Do not reuse the legacy ordered probe ladders.
- Migration across kernels uses an explicit old-to-new semantic mapping.
  Preserve the original file and report every unresolved or ambiguous key.
  A new payload meaning needs a version change; reusing the envelope version
  cannot hide a change in naming semantics.
- Projected sketch geometry stores the source reference plus a compact analytic
  descriptor. Martensite supplies analytic projection, including required
  extensions. A display polyline is only a cache. A lost source remains visibly
  unhealthy; it does not become an independent editable curve without Break Link.

## Alternatives and consequences

Pointer identity, face iteration order, and nearest-geometry matching are cheap
but change meaning after edits or on symmetric parts. Reject them as persistent
names. Provenance requires work in every relevant Martensite operation and may
block feature completion. Explicit external geometric selection remains useful
for import, but it has weaker evidence and must report that difference.

## Legacy disposition

Consolidate Oblikovati ADR-0040 and 0043, identity portions of 0006/0035/0057,
and the reference portion of 0055 at the
[fixed revision](legacy-map.md#source-baseline). Retain external descriptors,
provenance, analytic projection, and multi-parent history. Replace name-based
feature namespaces, suffix scans without uniqueness proof, ordinal fallbacks,
and geometric recovery behind a failed key. Legacy "shipped" notes do not
remove the uniqueness and namespace acceptance gates.

## Validation required

Use Go `model/identity/encoding.go`, `kernel/topo/provenance.go`, reference
binders, projection codecs, and edit/replay fixtures. The R1/R3 proof attaches
a dependent feature to an extrusion face, then edits, saves, reopens, undoes,
and redoes it. Extend with split/merge/deletion, repeated components, symmetric
geometry, imported descriptors, and malformed keys. Compare intended entities,
not just key lengths or whole-body volume. Versioned ordering tests include
the concern recorded in legacy issue #3069.
