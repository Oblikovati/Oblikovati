# Sketch reference API and entity order

This step implements `sketch.referenceKey` and `sketch.resolveReference` from
`Oblikovati.API` 0.157.0 for the current part or assembly document. It implements
[R3 issue #3765](https://github.com/Oblikovati/Oblikovati/issues/3765).

`DocumentSketchReferenceIndex` builds one checked catalogue from the current
immutable document snapshot and its internal document GUID. Sketch and entity
keys use the existing UUID-v5 namespace and name encoding, tested against the
saved Go values from commit `aa8297d1b0ac33c024f19e54821536d76e8a447a`.
Every inserted key must resolve to its target. Duplicate keys cause a named
ambiguity error before a lookup can choose an owner.

The API uses the canonical method constants, DTOs, and typed client. A selected
sketch key resolves to its current list index. An entity key also returns its
permanent entity ID and current owning sketch index. Delete and undo change the
current catalogue. Unknown, deleted, foreign, and noncanonical text returns
`found: false`. These queries do not solve geometry, edit history, or dirty a
document. A reference proves current ownership; it does not prove that a feature
or an associative geometry source is healthy.

Ordinary save, Save As, and reopen retain the internal GUID and reference keys.
Save Copy retains local sketch and entity IDs but creates a new document GUID.
The copy resolves only keys from its new namespace. Each query builds its
catalogue from the active owner, so a closed or inactive document cannot supply a
matching target.

`Sketch2D` now retains entity insertion order across points, lines, circles, and
arcs. Successful edits append once; failed edits leave the order unchanged.
Recipe equality and solver candidates retain this authored order. Document
entity enumeration uses this one sequence instead of reconstructing kind groups.

Planar recipe version 2 adds `entity_order`. Restore requires an exact permutation
of the owned entities: missing, duplicated, and foreign IDs are refused. Version
1 remains readable and restores its original point/line/circle/arc grouping.
Recipes whose authored order matches that legacy grouping still use version 1
and omit the new field. This keeps unchanged legacy files stable while ensuring
an old reader refuses a mixed-order version-2 recipe. The enclosing document and
placed-sketch schemas do not change.

Tests cover mixed geometry order, invalid permutations and versions, failed
edits, solver candidates, ambiguity, Go UUID values, current indices after delete
and undo, read-only hosts, part and assembly ownership, invalid requests, native
YAML round trips, save/reopen, Save As, and copied namespaces. The corresponding
Go router tests pass on the pinned baseline.

Entity presentation and movement classification still need the solver's actual
coordinate freedoms. Native picking, topology references, the UI, and Martensite
integration remain required for the complete application port.
