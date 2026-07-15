# Milestone 4 — Chorded-Rim Torus-Band Tessellator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Teach the torus band tessellator to mesh a chorded/mixed-edge footprint rim as a trimmed band (not the full donut), then unblock the torus setback runout deferred out of M3 — greening **T1** and making **T4** faithful (and **T9** as it routes through), corpus 53→54+.

**Architecture:** A surgical, additive change to ring recognition in `kernel/ops/closed_band_loft.go` — chain the boundary's non-seam edges into closed point-rings (mirroring the proven `traceClosedRings`/`bandWrapRings` pattern), feeding the existing `loftRows` unchanged; watertight by weld-points. Then admit `geom.Torus` in the M3 `setbackBossesFaithful` whitelist and lift the `bodyHasFragileBand` runout deferral (scoped so the obstacle path is byte-identical). The RailLoop corner engine, `assembleBody` weld, and the M3 setback path are otherwise unchanged.

**Tech Stack:** Go (GPL-2.0-only module `oblikovati`), `kernel/ops` B-rep tessellation + fillet engine; DRAWEXE 8.0.0 oracle.

**Design source (READ before implementing):** `docs/superpowers/specs/2026-07-15-chorded-rim-torus-band-tessellator-design.md`. Code map facts are inlined per task.

## Global Constraints

- **NO PR until the whole corpus is green.** Accumulate + commit per task on `feat/occt-blend-parity-corpus`.
- **Corpus non-regression, EVERY task:** `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` stays **≥ 53**, byte-identical on untouched cases (the `-v` is REQUIRED or grep counts 0). Greening tasks only ADD passes.
- **Tessellation regression tripwires (must stay green at every task):** `TestRimFilletTorusBand`, `TestRimFilletWatertightAcrossSizes` (`kernel/ops/fillet_rim_test.go`), `TestP2TorusBandNotFullDomain` (S9/T1/T3/T4, `model/feature/occtparity/fillet_p2_torus_band_test.go`), `TestIsFullCircleArc` (`kernel/ops/closed_band_loft_test.go`), the half-space/mass torus-volume tests (`halfspace_torus_side_test.go`, `massprops_orientation_test.go`).
- **Additive/gated:** the chaining path engages ONLY when the existing single-edge ring recognition yields < 2 rings. Plain rim fillets (single `geom.Circle`/full-sweep `geom.Arc3d` rings) MUST take the current code path byte-identically.
- **Oracle gates (area within OCCT 1%, `checkprops`):** T1 **15179.9** → [15028.1, 15331.7] (torus wall ≈1144.04 intact, NOT the 3947.84 donut); T4 **19514.7** → [19319.6, 19709.8] (torus ≈2826.04); torus-band mesh area within ≈1% of the analytic partial band `2π·r·(R·π/2 ± r)`.
- **Style:** funcs 4–20 lines; files < 500; explicit types; early returns; ≤ 2 indent; no duplication; error messages carry offending value + expected shape.
- **Tolerances model-relative** (`ResolutionForPoints(...).Weld()`, ADR-0042); never a bare `1e-6` in production. Seam-angular / weld tolerances reuse the existing `seamAngularTol`.
- **SPDX** `// SPDX-License-Identifier: GPL-2.0-only` on every new `.go`.
- **DRAWEXE oracle:** `../occt-build/lin64/gcc/bin/DRAWEXE`, env `test-utilities/occt-blend/oracle/drawenv.sh`, `printf 'source X.tcl\n' | DRAWEXE -b`.

## File Structure

- `kernel/ops/closed_band_loft.go` (modify) — `bandRingsAndSeam` gains a chained-ring fallback + robust seam ID; a new `chainBoundaryRings` helper (or reuse an extracted `traceClosedRings`). Watch the 500-line limit — if it grows too big, split the chaining into `kernel/ops/band_ring_chain.go`.
- `kernel/ops/band_ring_chain.go` (likely new) — the edge-chaining ring tracer + robust seam identification, curve-type-agnostic.
- `kernel/ops/closed_band_loft_test.go` / new `band_ring_chain_test.go` (modify/new) — synthesized chorded-band unit tests.
- `kernel/ops/fillet_setback_detect.go` (modify) — admit `geom.Torus` in `setbackBossesFaithful`.
- `kernel/ops/fillet_runout_faces.go` (modify) — lift/scope the `bodyHasFragileBand` runout deferral.
- `model/feature/occtparity/*` (modify) — add the setback+torus area gate; add T9 to the mesh-area set.

---

## Task 0: Spike — characterize the real T1/T4 setback torus boundary (disposable)

**Purpose:** the ring tracer + tests must match the real chorded-boundary shape. Gather facts; write NO production code; leave `git status` clean.

**Files:** throwaway probes in `kernel/ops/` (deleted after) + `experiments/` if useful.

- [ ] **Step 1:** Temporarily admit `geom.Torus` in `setbackBossesFaithful` and bypass the `bodyHasFragileBand` runout deferral, run the M3 setback path on the **T1** body (`fixtures/simple/T1.step`, R=8, edge-mid (0,−30,0)), and extract the intact torus wall face. Record: (a) how many boundary edges the torus face has and each edge's `geom` curve type (expect: host-side `Arc3d` pieces + patch-side `LineSegment` chords on the footprint rim; a single `Circle`/full-`Arc3d` on the opposite ring; one partial-`Arc3d` tube seam); (b) **does the tube seam survive as a single edge**, and is any footprint arc piece longer than it (the seam-mis-ID risk); (c) how many disjoint closed rings the non-seam edges chain into (must be 2); (d) the current tessellated torus-face area (confirm ≈**3947.68** = the RED baseline this milestone fixes).
- [ ] **Step 2:** Repeat for **T4** (R=8, edge-mid (0,−30,0)) — record the torus area (expect ≈**13816.9** full donut) and boundary shape.
- [ ] **Step 3:** Determine whether **T9** routes through the setback path (does it have a torus crossing-boss runout?) — record its boss types + whether the setback path fires.
- [ ] **Step 4:** Revert ALL probe changes (`git status` clean; `go build ./kernel/...` green; corpus 53). Write the facts to `.superpowers/sdd/m4-spike.md`: the seam edge's identity + how to distinguish it from footprint arcs (the robust seam rule), the ring piece counts, and the confirmed RED areas. **These facts parameterize Task 1's tracer + tests.**

**No commit** (spike is investigation).

---

## Task 1: Edge-chaining torus-band ring tracer (the tessellator fix)

**Problem:** `bandRingsAndSeam` (`closed_band_loft.go:67`) recognizes a ring only as a single `geom.Circle`/full-sweep `geom.Arc3d` edge; a chorded/mixed-edge footprint rim is silently dropped → `len(rings)==1` → `fullDomainGridMesh` → full donut. Fix: chain the boundary's non-seam edges into closed point-rings, curve-type-agnostic, gated additive.

**Files:**
- Create: `kernel/ops/band_ring_chain.go`
- Modify: `kernel/ops/closed_band_loft.go` (`bandRingsAndSeam` — chained fallback + robust seam ID)
- Test: `kernel/ops/band_ring_chain_test.go`

**Interfaces:**
- Consumes: `TessellateEdge(e, q)`/`discretizeEdge` (`edge_discretize.go:21`), `ResolutionForPoints(...).Weld()`, `loftRows`/`orderedRing` (`closed_band_loft.go`), the `traceClosedRings` chaining idea (`periodic_nurbs_mesh.go:151`).
- Produces: `chainBoundaryRings(f *topo.Face, q Quality) (rings [][]math.Point3, seamN int, seamMid math.Point3, ok bool)` — same return shape as `bandRingsAndSeam`, but builds rings by chaining non-seam edges; `ok=true` only when exactly 2 closed rings + a valid seam are recovered. `bandRingsAndSeam` tries the existing single-edge path first, then this fallback.

- [ ] **Step 1: Write the failing test** — synthesize a chorded torus band from a plain one and assert it meshes to the band area, not the donut. `kernel/ops/band_ring_chain_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import ("math"; "testing")

// choredTorusBandFace builds a plain rim-fillet torus band (brep.SolidCylinderFilletedTop,
// the same body TestRimFilletTorusBand uses), then subdivides its footprint-rim edge into n
// straight chords (via the same edgeInserts/subdivide machinery the setback path uses), so the
// face's rim is a chain of geom.LineSegment sub-edges — mimicking buildSetbackFaces output.
// Named helper (not inline), returns the intact torus *topo.Face + the analytic band area.
func choredTorusBandFace(t *testing.T, majorR, minorR float64, n int) (*topo.Face, float64) { /* Task 0 informs the exact construction */ }

func TestChainBoundaryRings_ChordedTorusMeshesAsBand(t *testing.T) {
	f, wantArea := choredTorusBandFace(t, 20, 5, 26) // T1's torus: R=20, r=5, ~26 chords
	m := TessellateFace(f, DefaultQuality())
	got := meshArea(m) // sum of triangle areas (reuse the corpus area helper)
	// full donut = 4π²·20·5 = 3947.84; the trimmed 90° band ≈ 2π·5·(20·π/2+5) = 1144.04
	if math.Abs(got-wantArea)/wantArea > 0.01 {
		t.Fatalf("chorded torus band meshed to area %.3f, want ≈%.3f (±1%%) — a full-donut %.1f means the chained-ring fallback did not engage", got, wantArea, 4*math.Pi*math.Pi*20*5)
	}
	if !meshIsWatertight(m) {
		t.Fatalf("chorded torus band mesh is not watertight")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails** — `go test ./kernel/ops -run TestChainBoundaryRings -v` → FAIL (meshes to ≈3947 donut; `chainBoundaryRings` undefined). Expected.

- [ ] **Step 3: Implement `chainBoundaryRings`** in `band_ring_chain.go`. Algorithm (mirror `traceClosedRings`): (1) discretize every boundary edge to an oriented polyline (`TessellateEdge`, curve-type-agnostic); (2) **identify the tube seam** by the robust rule from Task 0 — the meridian edge whose removal leaves the rest chaining into exactly 2 closed rings (topological), cross-checked geometrically (spans the tube parameter, not the ring parameter); (3) chain the remaining polylines head-to-tail by welded endpoints (`ResolutionForPoints(...).Weld()`) into closed rings; (4) return `ok=true` iff exactly 2 rings + a seam of ≥ 2 points. Keep each func 4–20 lines.

- [ ] **Step 4: Wire the fallback into `bandRingsAndSeam`** — try the existing single-edge recognition first; if it yields `len(rings) < 2`, call `chainBoundaryRings` and use its result if `ok`. The single-edge path stays byte-identical for plain rings.

- [ ] **Step 5: Run the test to verify it passes** — `go test ./kernel/ops -run TestChainBoundaryRings -v` → PASS (area ≈1144 ±1%, watertight).

- [ ] **Step 6: Byte-identity + tessellation tripwires** — `go test ./kernel/ops -run 'TestRimFilletTorusBand|TestRimFilletWatertight|TestIsFullCircleArc' -v` PASS; `go test ./model/feature -run TestP2TorusBandNotFullDomain -v` PASS (S9/T1/T3/T4 — the plain path unchanged). Corpus: `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` → **53** byte-identical. `go build ./kernel/... && go vet ./kernel/ops/ && golangci-lint run ./kernel/ops/...` clean.

- [ ] **Step 7: Commit** — `feat(tessellate): chain a chorded torus-band rim into a ring (fixes full-donut mesh)`.

---

## Task 2: Unblock the torus setback runout — green T1, make T4 faithful

**Problem:** with the tessellator fixed, admit torus bosses to the M3 intact-setback path. `setbackBossesFaithful` currently admits Cylinder/Cone/EllipticalCylinder only; `bodyHasFragileBand` defers torus/b-spline BODIES (a leftover from the OLD split path — the intact path no longer needs it). Scope the deferral so the shared obstacle path stays byte-identical.

**Files:**
- Modify: `kernel/ops/fillet_setback_detect.go` (`setbackBossesFaithful` — admit `geom.Torus`)
- Modify: `kernel/ops/fillet_runout_faces.go` (scope the `bodyHasFragileBand` runout deferral)
- Test: `kernel/ops/fillet_setback_close_test.go` (setback+torus area gate) + `model/feature/occtparity/fillet_p2_torus_band_test.go` (add T9 to the mesh-area set)

- [ ] **Step 1: Failing gate + faithfulness test** — `go test ./model/feature -run 'TestOCCTBlendSimple/T1$' -v` → FAIL (baseline +1.02%). Add `TestFilletEdges_T1TorusIntact` asserting the T1 result carries the torus wall as a SINGLE un-split face of area ≈**1144.04** (±11, NOT 3947.84) and the r6 cyl (565.487); mirror `TestFilletEdges_S4WallsIntact`.

- [ ] **Step 2: Verify it fails** — `go test ./kernel/ops -run TestFilletEdges_T1TorusIntact -v` → FAIL (torus meshes wrong or the path defers). Expected.

- [ ] **Step 3: Admit torus in the whitelist** — add `geom.Torus` to `setbackBossesFaithful`. Keep the per-type structure (one case per proven type).

- [ ] **Step 4: Scope the `bodyHasFragileBand` runout deferral** — `bodyHasFragileBand` is called by BOTH `collectRunouts` and `collectObstacles`. Remove/narrow it in the RUNOUT path only (the intact-setback path now handles torus; `setbackBossesFaithful` per-boss-type gating is the real gate), leaving `collectObstacles` untouched. PROVE the obstacle path is byte-identical: the obstacle-greened corpus cases must not move (diff the corpus name set).

- [ ] **Step 5: GREEN T1 + oracle gate** — `go test ./model/feature -run 'TestOCCTBlendSimple/T1$' -v` → PASS within **[15028.1, 15331.7]**; `TestFilletEdges_T1TorusIntact` PASS (torus ≈1144 single face). Do NOT loosen the gate; debug against the T1 DRAWEXE oracle if area is off.

- [ ] **Step 6: T4 faithful via the new path** — `TestOCCTBlendSimple/T4$` PASS within **[19319.6, 19709.8]** now through the setback path (torus ≈2826.04 intact) — add `TestFilletEdges_T4TorusIntact` (torus 2826.04±28 + cyl-r10 942.478). T4 must be faithful, not baseline-luck.

- [ ] **Step 7: Setback+torus area gate** — add a `kernel/ops` test that fillets a box with a crossing torus boss through the intact setback path and asserts the torus wall face tessellates to ≈1144 (NOT 3947) and the body mesh is watertight (the seam no existing test crosses).

- [ ] **Step 8: Corpus non-regression** — simple grid → **54** (S1/S4/T7/T1 + faithful T4), byte-identical elsewhere; broad grid no new failures. `TestP2TorusBandNotFullDomain` still green. Build/vet/lint clean.

- [ ] **Step 9: Commit** — `feat(blend): green T1 + faithful T4 runout (intact torus survivor)`.

---

## Task 3: T9 (+ any other setback-torus case) + milestone verification

**Files:** Modify `model/feature/occtparity/fillet_p2_torus_band_test.go` (add T9 to `p2TorusBandCases` mesh-area set); any whitelist/detection extension T9 needs.

- [ ] **Step 1:** From Task 0's finding on T9: if T9 routes through the setback path, add `TestFilletEdges_T9TorusIntact` + its `TestOCCTBlendSimple/T9$` oracle gate; add **T9 to the P2 mesh-area set** (`p2TorusBandCases`) so it is area-gated, not just validity-gated. If T9 is NOT a setback-torus case, document that in the commit and just add the T9 mesh-area gate for its existing (now-correct) tessellation.
- [ ] **Step 2:** Green T9 (or confirm it was already green and now area-gated). `go test ./model/feature -run 'TestOCCTBlendSimple/T9$' -v` PASS within OCCT 1%.
- [ ] **Step 3: Whole-milestone verification** — full corpus count (simple + broad), all tessellation tripwires, `go build ./kernel/... && go vet && golangci-lint run ./kernel/ops/...` clean. Diff the corpus name set vs base to confirm only T1/T4(/T9) moved.
- [ ] **Step 4: Commit** — `feat(blend): T9 torus runout + mesh-area gate`.

---

## Live test (pre-PR only — this milestone opens NO PR)

Per CLAUDE.md, before the eventual whole-corpus PR: drive `Oblikovati.AddIns.MCPBridge` to fillet a box with a crossing torus boss (a T1-like body), `Recompute`, MCP-screenshot — confirm the torus stands **intact** (no full-donut artefact) and the fillet fades smoothly. This milestone itself does not open a PR (the full corpus is not green).

## Notes for the executor

- Task 0's spike facts (seam identity, ring piece counts, RED areas) are load-bearing for Task 1 — read `.superpowers/sdd/m4-spike.md` before Task 1.
- The tessellator fix (Task 1) is decoupled from the runout (Task 2) via the synthesized chorded band, so Task 1 lands with NO red window; Task 2's wiring then greens T1/T4 end-to-end.
- Keep the chaining path strictly additive — a plain rim fillet must never enter it (byte-identity is gated by `TestP2TorusBandNotFullDomain` + `TestRimFilletTorusBand`).
- If the seam cannot be robustly identified, or a target case is not a clean 2-ring band, honest-reject (`ok=false`) so the caller keeps existing behavior — never fabricate a ring. Escalate rather than force.
