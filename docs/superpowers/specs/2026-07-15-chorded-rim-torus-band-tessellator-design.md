# Milestone 4 — Chorded-Rim Torus-Band Tessellator (design)

## Context

The M3 intact-boss runout re-architecture greens the fillet-runout family faithfully by
keeping each crossing boss wall **intact** and welding the fillet's setback patches to a
**chord-subdivided** copy of the boss footprint rim. For cylinder/cone/elliptical-cylinder
walls this tessellates correctly (they mesh as ruled strips). For a **torus** boss it does
not: the intact torus wall meshes as the **full donut** (`4π²Rr`) instead of the trimmed
band, so T1/T4 (and the torus setback subclass generally) fail the area gate and were
deferred out of M3.

**Root cause (mapped, `file:line`):** `bandRingsAndSeam` (`kernel/ops/closed_band_loft.go:67`)
recognizes a torus band's boundary ring **only** as a *single* edge whose curve is a
`geom.Circle`, or a `geom.Arc3d` with `|sweep|≈2π` (`isFullCircleArc`, `closed_band_loft.go:51`).
After the runout weld the intact torus footprint rim is no longer one edge — it is a **chain
of mixed edges**: host-side footprint arcs split at the wall seam (`hostSideFootArc`,
`fillet_setback_close.go:125`) plus patch-side straight chords. None of those match the switch,
so they are silently dropped, `bandRingsAndSeam` returns `len(rings)==1`, `closedBandLoftMesh`
bails (`closed_band_loft.go:30`), and `meshSeamCrossingFace` falls to `fullDomainGridMesh`
(`tessellate_trim.go:138`) — the whole `[0,2π]²` grid = the full donut. The file doc at
`tessellate_trim.go:22` names exactly this gap ("a seam-crossing periodic loop … needs a
constrained triangulation"). This is the **same bug class** as the earlier full-sweep-`Arc3d`
recognition gap fixed in `TestP2TorusBandNotFullDomain` — now with a chorded rim instead.

**Why it matters (CLAUDE.md):** tessellation correctness is the highest priority — the user
only ever sees the mesh; a full-donut torus corrupts area/mass/render/export. This gap blocks
the entire torus setback subclass.

## Goal

A **surgical** fix to torus band ring-recognition so a chorded/mixed-edge rim is chained back
into a ring and lofted correctly, then unblock the torus setback runout (T1/T4, and the torus
subclass — candidates T9/S9/T3 as they route through the setback path). Everything downstream
of ring recognition (`orderedRing` → `loftRows` → `stitchBandRows`) already works and is
curve-type-agnostic — it only needs the ring's **points**.

## Approach (chosen: A — surgical edge-chaining; B rejected)

**A — teach ring-recognition to chain edges.** Replace `bandRingsAndSeam`'s per-edge
type-switch with an edge-**chaining** ring tracer: pool the boundary edges, identify the tube
seam, and weld the remaining edges head-to-tail (by shared endpoints, curve-type-agnostic) into
closed **point**-rings. Feed the resulting rings into the **existing, unchanged** `loftRows`.

This pattern is already **proven in the codebase** and is the design precedent to mirror:
- `traceClosedRings` (`periodic_nurbs_mesh.go:151`) chains `nonSeamEdgePolylines` (each edge
  discretized via `discretizeEdge`, any curve type) head-to-tail by welded endpoints into closed
  rings — for the singly-periodic B-spline case. This is exactly the ring-recovery M4 needs.
- `bandWrapRings` (`saddle_band_loft.go:142`) pools several open rim edges into one ring via
  `dedupRingPoints` for developable saddle bands (currently `isDevelopableSide`-gated, never torus).

**Watertight by construction.** The rings are built from each edge's own tessellation points
(`TessellateEdge`/`discretizeEdge`, `edge_discretize.go:21`), which are the **exact weld
vertices** shared with the setback patches and host reconstruction (matched sampling, M3 Task 4).
So the lofted torus band welds to its neighbours point-for-point — no re-sampling of a smooth
circle that would crack the mesh at the chorded rim.

**B — a doubly-periodic covering-space CDT** (generalize `periodic_nurbs_cover.go` to two
periodic directions). More general (arbitrary torus trims), but heavier and higher-risk, and the
target cases are all 2-ring bands; other torus-trim shapes are already handled
(`torusComplementMesh`, `spiricBandMesh`). **Rejected** as more than the milestone needs; kept
only as a fallback if a target case proves not to be a clean 2-ring band.

### Design details

1. **Seam identification (the one subtle part).** Today the seam = the longest *partial*
   `Arc3d`. That heuristic is fragile once the footprint rim contributes its own partial arcs
   (`hostSideFootArc`) that may be longer than the tube seam. Use a **robust** rule: the tube
   seam is the meridian edge connecting the two rings — the edge whose removal leaves the
   remaining edges chaining into exactly **two** closed rings (topological), cross-checked
   geometrically (its supporting plane contains the torus axis / it spans the tube parameter,
   not the ring parameter). A spike (plan Task 0) verifies on T1/T4 whether the tube seam
   survives the setback rebuild as a single edge and how many pieces each ring is in.
2. **Ring chaining.** After removing the seam, chain all remaining edges by welded endpoints
   (model-relative `ResolutionForPoints(...).Weld()`, ADR-0042) into closed point-rings; require
   exactly 2. Reuse/extract the `traceClosedRings` chaining rather than re-implement it.
3. **Strict gating — byte-identity for the plain case.** The chaining path engages **only** when
   the existing single-edge recognition yields fewer than 2 rings. A plain rim fillet (two
   single-`Circle`/full-`Arc3d` rings) takes the current code path unchanged — proven by
   `TestRimFilletTorusBand` and `TestP2TorusBandNotFullDomain` staying byte-identical.
4. **Precedence preserved.** `bandRingsAndSeam` is reached only after `spiricBandMesh` and
   `torusComplementMesh` in `specialCurvedMesh` (`tessellate_trim.go:77`). The chaining must
   still yield exactly 2-rings-plus-seam only for a genuine band, and fall through (return
   `ok=false`) otherwise, so a spiric-cut or torus-complement face is never misclassified.

### Part II — unblock the torus setback runout (the payoff)

With the tessellator fixed, complete M3's deferred Task 7 (the runout logic is otherwise ready):
- Admit `geom.Torus` in the `setbackBossesFaithful` whitelist.
- Lift the `bodyHasFragileBand` deferral **scoped to the runout path only** (it exists to protect
  the OLD split path; the intact path no longer needs it) — proving the shared obstacle path stays
  byte-identical (the M3 Task 7 brief's approach).
- The torus footprint is a circle (DRAWEXE-confirmed) → `footprintConic` already handles it.
- Green T1 within **[15028.1, 15331.7]** (torus wall ≈1144.04 intact, NOT the 3947.84 donut) and
  make T4 faithful within **[19319.6, 19709.8]** (torus ≈2826.04). Extend to T9/S9/T3 if they
  route through the setback path.

## Verification & oracle gates

- **New area/watertight gate at the setback-boss + torus-band seam** — no existing test crosses
  it (setback tests check B-rep topology; torus-band tests use plain rim fillets). Add a test
  that fillets a box with a crossing torus boss (a T1-like body), runs the intact setback path,
  and asserts the torus wall face tessellates to ≈1144 (NOT ≈3947) and the body mesh is
  watertight. **Also add a mesh-area gate for T9** (currently checked only for solid *validity*
  in `TestB1ClosedSeamValidSolid`, not area in `p2TorusBandCases`).
- **Torus-band regression tripwires (must stay green):** `TestRimFilletTorusBand`,
  `TestRimFilletWatertightAcrossSizes` (`fillet_rim_test.go`), `TestP2TorusBandNotFullDomain`
  (S9/T1/T3/T4), `TestIsFullCircleArc`, the half-space/mass torus volume tests.
- **Corpus:** `TestOCCTBlendSimple` ≥ 53, byte-identical on untouched cases; then +T1(/T4/T9)
  as greened. Broad grid no new failures.
- **DRAWEXE oracle** per greened case (area within OCCT 1%), env
  `test-utilities/occt-blend/oracle/drawenv.sh`, `printf 'source X.tcl\n' | DRAWEXE -b`.
- **Live test (pre-PR only; this milestone opens no PR):** MCP-bridge fillet of a torus-crossed
  box + screenshot confirming the torus stands intact and the fillet fades smoothly.

## Risks & pitfalls

- **Seam mis-identification** (the footprint arc longer than the tube seam) — mitigated by the
  robust topological seam rule + the Task-0 spike.
- **Weld-tolerance sensitivity** — the chaining endpoint-weld and `dropClosingDup`
  (`closed_band_loft.go:91`) use `ResolutionForPoints(...).Weld()`; a change there touches the
  plain-circle rings too, so keep the chaining additive (new branch), never alter the existing
  ring path.
- **Ring point-count mismatch** between the chorded rim and the intact opposite ring — already
  handled by `stitchBandRows`/`zipUnequalRows` (unequal-row zip); confirm it holds for a very
  non-uniform chorded ring.
- **Genuine non-band torus faces** must still fall through (precedence with spiric/complement).
- **Honest-reject / do-no-harm** — if the chaining doesn't yield a clean 2-ring band, return
  `ok=false` so the caller keeps its existing behavior; never fabricate a ring.

## References (codebase precedents, not external)

`periodic_nurbs_mesh.go` (`traceClosedRings`, `nonSeamEdgePolylines`, `marchLoopUV`),
`periodic_nurbs_cover.go` (covering-space CDT, the rejected B), `saddle_band_loft.go`
(`bandWrapRings`), `closed_band_loft.go` (`bandRingsAndSeam`/`closedBandLoftMesh`/`loftRows` —
the reused loft), `tessellate_trim.go:14-23,44,124` (the entry + gap doc), `cdt_build.go`
(`constrainedTriangulationAll` — the primitive B would use). Prior instance of this bug class:
`model/feature/occtparity/fillet_p2_torus_band_test.go`.
