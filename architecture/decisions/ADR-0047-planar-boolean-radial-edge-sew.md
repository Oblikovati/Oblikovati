# ADR-0047 — Planar-boolean tangent-contact sew via Weiler radial-edge manifold extraction

**Status:** Accepted (2026-07-04). · **Resolves**
[Oblikovati#1726](https://github.com/Oblikovati/Oblikovati/issues/1726) (make the faceted-cylinder
line-tangency boolean exact, retire the nudge), the follow-up split from audit A4
([Oblikovati#1600], PR #1725). · **Builds on**
[ADR-0043](ADR-0043-generalized-provenance-naming.md) (edge/vertex provenance naming) and
[ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md) (model-relative tolerances). ·
**Touches:** `kernel/brep/boolean_radial_edge.go` (new pure core), `kernel/brep/boolean_mint.go`
(new naming/construction consumer), `kernel/brep/boolean_nonmanifold.go` (slimmed), and
`kernel/brep/boolean.go` / `boolean_stitch.go` (nudge removed). · **Supersedes** the
displaced-geometry nudge introduced for A4 in PR #1725.

## Context

A regularized boolean of two solids that touch along a **zero-measure locus** — a line or a point
— is a valid r-set (Requicha 1980) whose **boundary is non-manifold** along that locus. Our kernel
requires closed orientable 2-manifold bodies, so the sew must choose a manifold representation.

The old planar sew resolved a `>2`-edge-use tangent contact with two ad-hoc, tangled layers:
`resolveEdgeUses`/`pairTangentDihedrals`/`pickPartner` (edge level, with an unconditional
"prefer cross-operand" tie-break added in PR #1725 for the flush-seam fillet case) and a separate
`pinchedEndpoints` fan split (vertex level). On a **coplanar flush** overlap this was correct, but on
a **non-coplanar bowtie line-tangency** (a faceted-cylinder boss whose longitudinal edge grazes a
flat lug face — the 28BYJ-48 boss-on-lug) the cross-operand preference **fused the bowtie** into an
edge-manifold-looking body with **odd χ**. Unable to ship an invalid body, the boolean fell back to
**translating operand B by 0.1 µm** (`nudgeEps`) so the touch became a clearance — a recorded Defect
(`boolean.nudged-geometry`). Displaced coordinates poison flush mating, coplanar detection and
exports downstream (#1600). Naming and construction were interleaved into the same functions, so the
topology could not be corrected without risking the ADR-0043 reference keys.

The geometry-math-advisor established the correct model: pair the two boundaries of each **filled
angular wedge** around the edge — operand-agnostic; it reproduces cross-operand fusion at a coplanar
seam and same-operand real dihedrals at a bowtie — then cut each pinched vertex into per-disk
coincident duplicates. The team chose the **full Weiler radial-edge** rebuild over a surgical
tie-break patch (no shortcuts).

## Decision

Extract a **pure radial-edge core** (`radialSew → sewPlan`) that owns the manifold topology and
nothing else, with naming and construction as downstream consumers of a naming-free plan.

- **`radialSew(verts, faces, uses) sewPlan`** — a total function over a combinatorial `sewPlan`
  (`edgeGroup`s + a per-welded-vertex `vertexDisk` partition + a `(face,ring,pos)→group` map). It
  moves no coordinate, mints no entity, names nothing.
- **Filled-wedge pairing** — around a non-manifold edge the half-edge uses are azimuth-sorted and
  each **ENTER** boundary (a reversed use — the loop orientation already encodes the material side,
  `axis×interior·(−N) = ∓1` by travel sign) is paired with the next **EXIT** in azimuth, the two
  boundaries of one filled dihedral wedge. No operand preference.
- **Vertex disks** — the incident edge-groups are partitioned into radial disks (two groups share a
  disk iff a face uses both); every disk beyond the first mints a coincident duplicate vertex, so a
  line/point kiss separates into coincident manifold shells.
- **`mintEntities`** consumes the plan and holds ADR-0043 naming (parent-pair + rank) and
  `topo.Builder` construction.
- **The nudge is deleted.** A tangent contact that resolves to a valid manifold ships exact and
  records `boolean.tangent-contact` (Info); one that does not records `boolean.tangent-unresolved`
  (Defect) and returns the invalid body so the caller declines to CSG **observably** — geometry is
  never moved to force validity (SoS discipline, Edelsbrunner & Mücke 1990).

## Consequences

- **(+)** Every tangent/flush/bowtie contact ships **exact, undisplaced** coordinates as a valid
  2-manifold. A pure line kiss → two coincident shells (χ=4, volume exactly additive); a bowtie on
  an otherwise-connected body → one valid shell (boss-lug volume matches inclusion–exclusion to
  machine epsilon).
- **(+)** The manifold invariant is guaranteed by the sew, not gated after the fact; the radial
  topology is unit-testable without lineage; provenance stays isolated.
- **(+)** Every non-degenerate boolean and every coplanar-flush union is **byte-identical** (the
  Slice-0 characterization golden over V/E/F/χ + reference keys, plus the full brep/ops suites).
- **(−)** A pure line/point kiss now returns **two shells**, not one — the honest topology, but a
  semantics change callers must accept.
- **(0)** The anticipated risk that two coincident contact edges collide on their ADR-0043
  reference key (same parent pair *and* same position, which rank-along-line cannot separate) was
  investigated and **does not arise**: with same-operand filled-wedge pairing each coincident edge
  borders a *distinct* pair of faces (e.g. boss facet A∩lug vs boss facet B∩lug), so the parent
  pairs — and thus the keys — differ. Guarded by a key-uniqueness assertion in the certification;
  no companion naming change was needed.

## Rejected

- **The surgical tie-break** — gate the existing cross-operand preference on coplanarity and keep
  the two-layer ad-hoc sew. Rejected by the team as a shortcut: it patches the symptom without
  unifying the edge- and vertex-level resolutions and leaves the naming/topology tangle that caused
  the fragility. (Verified empirically insufficient: gating fusion still let `fallback` pair a boss
  facet to a lug half where the azimuth order interleaves the operands.)
- **A single non-manifold body** (OCCT compound style) — violates the kernel's 2-manifold invariant.
- **Keeping the nudge** — moves geometry; the defect #1600/#1726 exist to kill.

## References

Weiler (1988), "The radial edge structure for non-manifold geometric modeling." · Requicha (1980),
regularized set operations. · Mäntylä (1988), Euler–Poincaré & Euler operators. · Edelsbrunner &
Mücke (1990), Simulation of Simplicity (perturb only inside predicates). · geometry-math-advisor and
software-architect-advisor briefs recorded on #1726.
