# ADR-0057 — Reconstruction is the one boolean; brep.Boolean is deleted

**Status:** **Superseded by** [ADR-0058](ADR-0058-tolerant-analytic-boolean.md)
(2026-09-01, #2183). Its central decision — *reconstruction is the one boolean; `brep.Boolean`
is deleted* — is **reversed** on measured performance grounds: exact `big.Rat` mesh reconstruction for all
planar input ran ~1.5× slower and did not scale. `brep.Boolean` is **not** deleted; ADR-0058
promotes it to the planar core of the one general engine. Read this document as the record of a
direction that was tried and abandoned, not as guidance. (Originally: Accepted — building on
`m48/kernel-ground-rules`.) · **Scopes**
[Oblikovati#2247](https://github.com/Oblikovati/Oblikovati/issues/2247) (merge
`brep.Boolean`/`BooleanDiag`'s planar-only engine into the one general boolean) and unblocks the
M48/C2 boolean cluster ([#2248](https://github.com/Oblikovati/Oblikovati/issues/2248),
[#2250](https://github.com/Oblikovati/Oblikovati/issues/2250),
[#2251](https://github.com/Oblikovati/Oblikovati/issues/2251),
[#2252](https://github.com/Oblikovati/Oblikovati/issues/2252)). · **Builds on and supersedes the
"reconstruction is a curved-only fast path" stance of**
[ADR-0056](ADR-0056-analytic-face-reconstruction-boolean.md); **extends**
[ADR-0043](ADR-0043-generalized-provenance-naming.md) (provenance naming) with multi-parent face
identity. · **Deletes:** `kernel/brep/boolean.go` `Boolean`/`BooleanDiag` (the planar-only exact
engine), its `exactTangentIsValid` gate (also [#2293](https://github.com/Oblikovati/Oblikovati/issues/2293)),
and the `analyticFaceCount==0` planar carve-out in `kernel/ops/meshbool_recovery.go`. · **Touches:**
`kernel/ops/meshbool_reconstruct*.go`, `kernel/ops/boolean.go`, `kernel/brep/reconstruct_boolean.go`,
`kernel/ops/meshbool_recovery.go`, and the downstream planar-boolean corpus goldens.

## Context

ADR-0056 made analytic reconstruction the curved-boolean path but kept it **gated off for all-planar
operands** (`analyticFaceCount(target)+analyticFaceCount(tool) == 0` → decline), leaving
`brep.Boolean`/`BooleanDiag` as a second, planar-only exact engine. Two exact boolean engines is the
exact ground-rule violation M48/C2 exists to remove ("A generalization is complete only when the
special cases it replaces are deleted"; "one general pipeline per operation").

A 2026-08-29 probe (recorded on #2247) established the real state after the arrangement-hang fix
(`5354c35f`):

- Reconstruction is **geometry-correct on planar operands**: a matrix of planar unions/differences/
  intersections each reconstructs to a *valid solid with the correct volume*.
- The ONLY divergence from `brep.Boolean` is **face structure**: reconstruction *merges coplanar
  faces* (an L-prism union is 8 faces — the minimal, Parasolid/Inventor-equivalent topology), where
  `brep.Boolean` leaves the operands' coplanar fragments split (14 faces).
- That difference **compounds through sequential feature booleans**: chamfering three edges then the
  corner of a box, each cut reconstructs cleanly, but by the corner-blend cut the body is already
  reconstruction-shaped and the corner face splits across the new complex vertex; both engines decline
  it and the faceted CSG fallback fragments the corner into 8 triangles instead of 1.

So removing the guard today regresses exactly four `model/feature` tests
(`TestChamferMiterEdgeIsFlush`, `TestChamferFlatCornerBlendsAsymmetric`, `TestFullRoundRoundsRibTop`,
`TestCornerSeamOverlapAddsProudTab`) — not because reconstruction is wrong, but because downstream
naming, goldens, and the faceted fallback are all calibrated to `brep.Boolean`'s split-face topology.

## Decision

**Reconstruction becomes the one boolean. `brep.Boolean` is deleted. Downstream is re-baselined to
the merged-coplanar topology.** The merged-coplanar result is the correct, commercial-kernel end state;
we do not degrade reconstruction to match the old engine.

Merging coplanar faces from *different operands* changes face identity, so ADR-0043 is extended with
**multi-parent face identity**: a face produced by fusing N coincident-surface operand faces
(`mergeCoincidentTags`) carries the reference key of *every* parent, and the guarded resolver
(`FacesByKey`) resolves any parent's key to the merged face. This satisfies the naming ground rule "a
pick must survive the operation that consumed it" in both directions — a pick on either operand's
coplanar face survives the merge.

### Mechanism

1. **Carry the parent set through the merge.** `mergeCoincidentTags` already union-finds coincident
   refs; today only the representative `faceSurfaceRef.face` survives. Track the full member set per
   group and thread it to reconstruction as a `[]topo.Lineage` (parent lineages) on `brep.ReconInput`
   alongside the single representative `Lineage`.
2. **Register every parent key.** `brep.ReconstructBooleanBody` registers the built face under each
   parent lineage's key (verify-on-write round-trip per ADR-0043: store, re-resolve, reject a key that
   does not round-trip to the same face).
3. **Clean fallback topology.** When reconstruction legitimately declines (a genuinely Euler-
   inadmissible split it cannot yet close), the faceted fallback must emit **merged coplanar faces**,
   not triangle soup, so a declined case still yields clean face topology (this is what fragments the
   chamfer corner today). Coplanar-facet merge is a single operation shared by both paths.
4. **Delete the planar carve-out and `brep.Boolean`.** Remove the `analyticFaceCount==0` guard; route
   every boolean through reconstruction; delete `kernel/brep` `Boolean`/`BooleanDiag`/
   `exactTangentIsValid` and update the one production caller (`kernel/ops/boolean.go:174`) and the
   `kernel/brep` drill/boss family that call the planar engine directly.
5. **Re-baseline the corpus.** Update the planar-boolean goldens and face-count assertions to the
   merged-coplanar topology; every re-baselined body is re-asserted valid (`ops.Validate`) and
   per-face against the analytic oracle, never whole-body volume alone.

### Sequencing (each a committable increment on `m48/kernel-ground-rules`)

1. This ADR.
2. Multi-parent lineage plumbing (`faceSurfaceRef` group members → `ReconInput.Parents` →
   `ReconstructBooleanBody` multi-key registration) + a naming round-trip test proving both parents'
   keys resolve to a merged face. Behaviour of the result geometry is unchanged; only identity is
   added, so no goldens move.
3. Coplanar-facet merge shared by the faceted fallback (fixes the chamfer-corner fragmentation
   independently of the guard).
4. Remove the guard, re-baseline the four regressions + the planar corpus, gate per-face.
5. Delete `brep.Boolean`/`BooleanDiag`/`exactTangentIsValid`; close #2247, #2293, and the cluster.

## Consequences

- **Positive:** one exact boolean pipeline; merged-coplanar (commercial-kernel) topology; the C2
  boolean cluster and #2293 close; net −1 exact engine, −1 manifold check, −1 fallback carve-out.
- **Cost:** a corpus re-baseline (planar-boolean goldens + face-count assertions move once) and the
  multi-parent identity capability. The re-baseline is a one-time, per-face-verified migration.
- **Risk:** face identity is now many-to-one at a coplanar merge; the verify-on-write round-trip and a
  dedicated multi-parent naming corpus guard it. Reconstruction declines that still fall to the faceted
  path now yield clean (merged) topology, so a decline is a tolerance/analytic-fidelity regression, not
  a topology regression.
