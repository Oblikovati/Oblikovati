# ADR-0021: Versioned semantic topology reference keys

Date: 2026-10-13. Status: Accepted for the Rust target design.
This specifies the new payload required by ADR-0005 without changing legacy keys.

Use `OKRK` scheme 2 for Rust semantic topology names. Scheme 0 and scheme 1 retain
Go payload meaning and canonicalize to scheme 1. Scheme 2 stays scheme 2 during
encoding and exact binding. Never reinterpret a Go lineage payload as a Rust name.
This removes the scheme-1-only codec restriction and replaces the unnamed prism
query boundary with verified semantic body and face names. No geometry engine,
old-key fallback, or parallel naming lookup is added.

The envelope context is a nonzero stable feature recipe ID. Its kind uses the
model identity numbering (face 1, body 4), not Go kernel/topo numbering. The
scheme-2 payload contains the document GUID's 16 RFC UUID bytes, an operation tag,
a role tag, and ordered nonzero source recipe IDs. Tags use lowercase ASCII
letters, digits and hyphens, with 1..64 bytes each. Each tag has a little-endian
u16 byte count. Sources have a little-endian u16 count (0..64), followed by
little-endian u64 IDs. Trailing bytes and unsupported shapes are refused.

The selected sketch prism operation is `sketch-prism`. Roles `body`, `start`, and
`end` each carry the source sketch ID. Role `side` carries sketch ID then source
line ID. Cap roles mean the kernel sweep's first and last shapes, including a
negative or oblique displacement. Names do not depend on coordinates, display
names, face order, or memory addresses. Construction history must cover each
result face exactly once. A written name must bind back to that exact entity.

This format is deliberately distinct from Go's display-name-based lineage.
Reusing scheme 1 would risk false binding. Old-to-new migration requires an
explicit semantic correspondence and remains separate work. Operation tags and
role meanings are stable; a changed meaning requires a new scheme. Split/merge,
healing, occurrence publication, and unnamed edge/vertex derivation still need
complete provenance under ADR-0005. This change does not claim those operations.

Validation includes exact format bytes, legacy non-reinterpretation, malformed
payloads, duplicate names, wrong namespaces, per-face analytic queries, depth
and profile edits, input traversal changes, holes, negative/oblique sweeps,
save/reopen and undo invalidation. The existing OCCT prism oracle remains the
geometry gate. No recognizer, tolerance constant, geometric fallback, or native
type assertion is added by naming.
