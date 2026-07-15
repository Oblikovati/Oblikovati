# Milestone 4 — Chorded-Rim Torus-Band Tessellator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Green the two torus setback-runout cases — **T1** and **T4** — by (1) making the M3 setback rebuild reconstruct the intact torus wall's footprint rim as a full 360° loop, and (2) teaching the band tessellator to mesh that chorded/mixed-edge rim as a trimmed band (not the full donut). Corpus 53→54.

**Architecture:** Two surgical, additive fixes plus a wiring step. First, `kernel/ops/fillet_setback_close.go` gets a **scale-invariant footprint-rim partition** (the current minor-vs-major arc heuristic is tuned for small cylinder footprints and produces a malformed 118° out-and-back rim on the large r=25 torus footprint). Then `kernel/ops/closed_band_loft.go` gets an **edge-chaining ring tracer + congruence gate** so a chorded rim lofts correctly and a malformed rim is honest-rejected. Finally, `geom.Torus` is admitted to the M3 `setbackBossesFaithful` whitelist and the `bodyHasFragileBand` runout deferral is scoped, routing T1/T4 through the now-correct path. The RailLoop corner engine, `assembleBody` weld, and the M3 setback detect/extract stages are unchanged.

**Tech Stack:** Go (GPL-2.0-only module `oblikovati`), `kernel/ops` B-rep tessellation + fillet engine; DRAWEXE 8.0.0 oracle.

**Design sources (READ before implementing):**
- `docs/superpowers/specs/2026-07-15-chorded-rim-torus-band-tessellator-design.md` — the original design.
- `.superpowers/sdd/m4-spike.md` — the Task-0 spike facts (the rim is malformed; edge inventory; RED areas; seam-ID rule). **Load-bearing for Tasks 1 & 2.**
- `.superpowers/sdd/m4-rim-partition-derivation.md` — the geometry-math-advisor derivation of the scale-invariant σ-partition (Task 1's algorithm) + the ring-congruence gate (Task 2). **Load-bearing for Tasks 1 & 2.**

## Global Constraints

- **NO PR until the whole corpus is green.** Accumulate + commit per task on `feat/occt-blend-parity-corpus`.
- **Corpus non-regression, EVERY task:** `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` stays **≥ 53** and byte-identical on untouched cases through Tasks 1–2, then **54** after Task 3 (the `-v` is REQUIRED or grep counts 0). Greening only ADDS passes; only T1/T4 may move.
- **Shared-substrate proof (Task 1):** the rim partition is shared code that makes S1/S4/T7 faithful. Any change MUST be proven corpus-neutral for cyl/cone/ellipse by **revert-and-diff** (revert the change in place, confirm the identical PASS/FAIL corpus set, restore) — the M3 Task-4/Task-6 discipline.
- **Tessellation regression tripwires (green at every task):** `TestRimFilletTorusBand`, `TestRimFilletWatertightAcrossSizes` (`kernel/ops/fillet_rim_test.go`), `TestP2TorusBandNotFullDomain` (S9/T1/T3/T4, `model/feature/occtparity/fillet_p2_torus_band_test.go`), `TestIsFullCircleArc` (`kernel/ops/closed_band_loft_test.go`), the half-space/mass torus-volume tests (`halfspace_torus_side_test.go`, `massprops_orientation_test.go`).
- **Additive/gated:** the ring-chaining path (Task 2) engages ONLY when single-edge ring recognition yields < 2 rings; plain rim fillets (single `geom.Circle`/full-`Arc3d` rings) MUST take the current code path byte-identically.
- **Oracle gates (area within OCCT 1%, `checkprops`):** T1 **15179.9** → [15028.1, 15331.7] (torus wall ≈1144.04 intact, NOT the 3947.68 donut); T4 **19514.7** → [19319.6, 19709.8] (torus ≈2826.04, NOT 13816.88); torus-band mesh area within ≈1% of the analytic partial band.
- **No red window:** Tasks 1 and 2 both land with the torus still deferred (corpus 53, byte-identical). Only Task 3's wiring flips T1/T4 — and by then both the rim fix and the tessellator are in place, so they go straight to green.
- **T9 is OUT of scope** — the spike proved it has no torus boss (its fragile band is a free-form `BSplineSurface`); it never routes through this path and needs a separate periodic-b-spline mesher.
- **Style:** funcs 4–20 lines; files < 500; explicit types; early returns; ≤ 2 indent; no duplication; error messages carry offending value + expected shape.
- **Tolerances model-relative** (`ResolutionForPoints(...).Weld()`, ADR-0042); never a bare `1e-6`. Angular tol on a conic = `k·res.Weld()/r_f` (chord-to-angle), `k≈2..4`, per the derivation's pitfalls.
- **SPDX** `// SPDX-License-Identifier: GPL-2.0-only` on every new `.go`.
- **DRAWEXE oracle:** `../occt-build/lin64/gcc/bin/DRAWEXE`, env `test-utilities/occt-blend/oracle/drawenv.sh`, `printf 'source X.tcl\n' | DRAWEXE -b`.

## File Structure

- `kernel/ops/fillet_setback_close.go` (modify) — replace the local minor-vs-major arc chooser (`hostSideFootArc`/`footEdgeward` and the `footprintMajorArc` bisector path) with the scale-invariant σ-partition in `bossRimSubArcs`; add a closure-invariant guard. Watch the 500-line limit; if it grows past, split the partition into `kernel/ops/fillet_setback_partition.go`.
- `kernel/ops/band_ring_chain.go` (new) — the edge-chaining ring tracer + robust seam ID + the ring-congruence gate, curve-type-agnostic.
- `kernel/ops/closed_band_loft.go` (modify) — `bandRingsAndSeam` gains the chained-ring fallback (gated on single-edge recognition yielding < 2 rings).
- `kernel/ops/fillet_setback_detect.go` (modify) — admit `geom.Torus` in `setbackBossesFaithful`.
- `kernel/ops/fillet_runout_faces.go` (modify) — scope the `bodyHasFragileBand` runout deferral so it no longer defers torus bodies on the runout path (obstacle path untouched).
- Tests: `kernel/ops/fillet_setback_close_test.go` (rim partition + faithfulness), `kernel/ops/band_ring_chain_test.go` (new), `model/feature/occtparity/fillet_p2_torus_band_test.go` (setback+torus area gate).

---

## Task 1: Scale-invariant footprint-rim partition (the setback-close rim fix)

**Problem (measured, Task-0 spike §CRITICAL):** the M3 setback rebuild reconstructs the intact torus wall's footprint rim via `bossRimSubArcs` → `hostSideFootArc`/`footEdgeward` (`fillet_setback_close.go:281,313,332`), a **local minor-vs-major midpoint test** tuned for small cylinder footprints. On the large torus footprint (circle radius r_f=25) it returns the 118° minor arc for `hostA`, which `band`+`hostB` then re-cover — an out-and-back slit spanning only 118° of azimuth, 242° dropped. An intact wall's rim must be the full 360° conic. This is upstream of the tessellator: no mesh change can rescue a geometrically wrong rim.

**Fix (derivation `.superpowers/sdd/m4-rim-partition-derivation.md`, rule (b), §D2):** the two crossings are `cross1,cross2 = F ∩ L` (footprint conic ∩ fillet plane-contact line, both on `σ=0`). Partition in the conic's **native angular parameter**: `band` = the between-crossings interval whose interior midpoint has `σ>0` (edge-side); `hostA`/`hostB` = the complement `F ∖ band`, split at the seam — never chosen independently. Verify closure `Δ_hostA+Δ_band+Δ_hostB=2π`. Delete the minor/major bisector heuristic. This reproduces the verified small-cylinder behavior (band=major/host=minor there) and fixes the large-torus behavior (band=minor/host=major) with one σ-sign test evaluated deep inside the notch, never near `L`.

**Files:**
- Modify: `kernel/ops/fillet_setback_close.go` (`bossRimSubArcs` and helpers)
- Test: `kernel/ops/fillet_setback_close_test.go`

**Interfaces:**
- Consumes: `crossingBoss{wall geom.Surface; footEdge *topo.Edge; host *topo.Face; xSetback float64}` (`fillet_setback_detect.go`), `footprintSubArc`/`footprintConic`/`footprintCenter`, `filletContact`/`spineParam`, `ResolutionForPoints(...).Weld()`, the `EllipseFull` param path for T7's ellipse footprint.
- Produces: `bossRimSubArcs(boss, cyl, seam, cross1, cross2, bandInner)` returns the ordered `[]geom.Curve3` `{hostA, band…, hostB}` that now partition the full conic (Δ-sum 2π); `bossRimRing` unchanged in signature.

- [ ] **Step 1: Write the failing test** — a synthesized T1-scale torus boss whose rim must span the full circle. `kernel/ops/fillet_setback_close_test.go`:

```go
// synthTorusSetbackBoss builds a crossingBoss mimicking T1's intact torus wall: a torus
// wall surface (major R=20, minor r=5) fused so its host-plane footprint is a circle of
// radius r_f=25 centered at origin, the fillet R=8 band along the box edge at the bottom.
// Seam at azimuth 0° (world (25,0,0)); the fillet interference crosses the footprint at
// cross1 (−11.874,−22,0) [az −118.4°] and cross2 (11.874,−22,0) [az −61.6°]. Named helper.
// (Task-0 spike .superpowers/sdd/m4-spike.md gives the exact frame; construct the footEdge
// as a geom.Arc3d/Circle of radius 25, the host as a geom.Plane, and the fillet geom.Cylinder.)
func synthTorusSetbackBoss(t *testing.T) (boss crossingBoss, fillet geom.Cylinder, seam, c1, c2 math.Point3) { /* spike-informed */ }

func TestBossRimSubArcs_TorusFootprintSpansFullCircle(t *testing.T) {
	boss, fillet, seam, c1, c2 := synthTorusSetbackBoss(t)
	subs, ok := bossRimSubArcs(boss, fillet, seam, c1, c2, nil)
	if !ok {
		t.Fatalf("bossRimSubArcs rejected a valid torus footprint boss")
	}
	span := totalDirectedSpan(subs, boss.footEdge) // Σ directed native-param spans, reduced into (0,2π)
	if math.Abs(span-2*math.Pi) > 1e-3 {
		t.Fatalf("footprint rim spans %.1f°, want 360° — a partial span means the minor arc was chosen (the 118° slit bug); subs=%d", span*180/math.Pi, len(subs))
	}
	// hostA (seam→cross1) must be the MAJOR arc on this large footprint (≈241.6°, not 118.4°).
	if a := directedSpan(subs[0], boss.footEdge); a < math.Pi {
		t.Fatalf("hostA span %.1f° < 180° — chose the minor arc for the large torus footprint", a*180/math.Pi)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails** — `go test ./kernel/ops -run TestBossRimSubArcs_TorusFootprint -v` → FAIL (span ≈237° / hostA ≈118°, the current minor-arc bug), or `totalDirectedSpan` undefined. Expected.

- [ ] **Step 3: Implement the σ-partition** in `bossRimSubArcs` (rule (b), §D2). Steps: compute `ê` (edgeward unit, guarded) and `σ(p)=(p−p_L)·ê`; get native params `t_s,t_1,t_2` of seam/cross1/cross2 on the footprint conic; `band` = the (t_1,t_2) native interval whose midpoint has `σ>0`; verify `σ(seam)<0` and `t_s` in the host complement (else reseat to the band-midpoint antipode, §D4); emit `hostA = t_s→t_1`, `band = t_1→t_2` (carrying `bandInner`), `hostB = t_2→t_s`, each an exact `footprintSubArc`/`ellipseSubArc` on its native interval. Guard closure `Δ_hostA+Δ_band+Δ_hostB=2π` (angular tol `k·res.Weld()/r_f`); on failure return `ok=false` (honest-reject). Delete `hostSideFootArc`/`footEdgeward`/`footprintMajorArc` minor-vs-major logic. Keep each func 4–20 lines; add `totalDirectedSpan`/`directedSpan` helpers (used by the test and the closure guard). Handle the near-tangency (cross1≈cross2 → full-circle rim, no notch), full-engulfment, and >2-crossing honest-rejects per §Numerical pitfalls.

- [ ] **Step 4: Run the test to verify it passes** — `go test ./kernel/ops -run TestBossRimSubArcs_TorusFootprint -v` → PASS (span 360°, hostA major).

- [ ] **Step 5: Corpus-neutrality + revert-and-diff proof** — corpus stays **53** byte-identical (torus not yet admitted; S1/S4/T7 must produce the identical rim). Run `go test ./model/feature -run TestOCCTBlendSimple -v` and confirm the same PASS set; run `TestFilletEdges_S1WallsIntact`/`S4WallsIntact`/`T7` (or the current faithfulness tests) green. Then **prove neutrality by revert-and-diff**: revert the `bossRimSubArcs` change in place, capture the corpus PASS/FAIL set, restore, and confirm identical — recording the result. `go build ./kernel/... && go vet ./kernel/ops/ && golangci-lint run ./kernel/ops/...` clean.

- [ ] **Step 6: Commit** — `fix(blend): scale-invariant footprint-rim partition (full 360° torus rim)`.

---

## Task 2: Edge-chaining torus-band ring tracer + congruence gate (the tessellator fix)

**Problem:** `bandRingsAndSeam` (`closed_band_loft.go:67`) recognizes a ring only as a single `geom.Circle`/full-sweep `geom.Arc3d`; a chorded/mixed-edge footprint rim (30 `geom.LineSegment` chords after the Task-1 rim rebuild) is dropped → `len(rings)==1` → `fullDomainGridMesh` → full donut. Fix: chain the boundary's non-seam edges into closed point-rings, curve-type-agnostic, gated additive; and add a **congruence gate** (from the derivation) so a malformed non-congruent rim is honest-rejected rather than lofted into garbage.

**Files:**
- Create: `kernel/ops/band_ring_chain.go`
- Modify: `kernel/ops/closed_band_loft.go` (`bandRingsAndSeam` — chained fallback)
- Test: `kernel/ops/band_ring_chain_test.go`

**Interfaces:**
- Consumes: `TessellateEdge`/`discretizeEdge` (`edge_discretize.go:21`), `ResolutionForPoints(...).Weld()`, `loftRows`/`orderedRing` (`closed_band_loft.go`), the `traceClosedRings` chaining idea (`periodic_nurbs_mesh.go:151`).
- Produces: `chainBoundaryRings(f *topo.Face, q Quality) (rings [][]math.Point3, seamN int, seamMid math.Point3, ok bool)` — same shape as `bandRingsAndSeam`; builds rings by chaining non-seam edges; `ok=true` only when exactly 2 closed rings survive the congruence gate. `bandRingsAndSeam` tries single-edge recognition first, then this fallback.

- [ ] **Step 1: Write the failing tests** — (a) a synthesized *correct* 360° chorded torus band meshes to the band area; (b) a synthesized *malformed* (full-circle vs 118° out-and-back) ring pair is honest-rejected. `kernel/ops/band_ring_chain_test.go`:

```go
// choredTorusBandFace builds a CORRECT torus band (brep rim fillet, R=20 r=5), then
// subdivides its FULL-circle footprint rim into n straight chords (edgeInserts machinery),
// so the rim is a full-360° chain of geom.LineSegment sub-edges — mimicking the Task-1
// rebuilt rim. Named helper; returns the intact torus *topo.Face + the analytic band area.
func choredTorusBandFace(t *testing.T, majorR, minorR float64, n int) (*topo.Face, float64) { /* spike-informed */ }

func TestChainBoundaryRings_ChordedTorusMeshesAsBand(t *testing.T) {
	f, wantArea := choredTorusBandFace(t, 20, 5, 30)
	m := TessellateFace(f, DefaultQuality())
	got := meshArea(m)
	if math.Abs(got-wantArea)/wantArea > 0.01 {
		t.Fatalf("chorded torus band meshed to %.3f, want ≈%.3f (±1%%) — full-donut %.1f means the chained-ring fallback did not engage", got, wantArea, 4*math.Pi*math.Pi*20*5)
	}
	if !meshIsWatertight(m) {
		t.Fatalf("chorded torus band mesh is not watertight")
	}
}

// slitTorusBandFace builds a MALFORMED band: full-circle top rim vs a 118° out-and-back
// footprint slit (the pre-Task-1 defect shape) — used to prove the congruence gate rejects.
func slitTorusBandFace(t *testing.T) *topo.Face { /* spike-informed */ }

func TestChainBoundaryRings_RejectsNonCongruentRings(t *testing.T) {
	_, _, _, ok := chainBoundaryRings(slitTorusBandFace(t), DefaultQuality())
	if ok {
		t.Fatalf("chainBoundaryRings accepted a full-circle vs 118°-slit ring pair — the congruence gate must honest-reject (|U|/V≈0 on the slit ring), not loft garbage")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail** — `go test ./kernel/ops -run TestChainBoundaryRings -v` → FAIL (donut area; `chainBoundaryRings` undefined). Expected.

- [ ] **Step 3: Implement `chainBoundaryRings`** in `band_ring_chain.go` (mirror `traceClosedRings`): (1) discretize every boundary edge to an oriented polyline (`TessellateEdge`, curve-agnostic); (2) **seam = the unique meridian edge** — the only boundary edge whose endpoints differ in the tube parameter v (vSpan ≳ tol, both rings iso-v), cross-checked topologically (removing it leaves exactly 2 closed cycles); (3) chain remaining polylines head-to-tail by welded endpoints (`ResolutionForPoints(...).Weld()`) into rings; (4) **congruence gate** (derivation §"Tessellation congruence gate"): for each ring compute signed advance `U=Σunwrap(Δu)` and unsigned variation `V=Σ|Δu|` in the wall around-parameter; require `|U|/V ≥ 1−ε` (monotone, no out-and-back) and `|U_bot|≈|U_top|` with equal ranges; (5) return `ok=true` iff exactly 2 congruent rings + a seam of ≥ 2 points. Keep each func 4–20 lines.

- [ ] **Step 4: Wire the fallback into `bandRingsAndSeam`** — try the existing single-edge recognition first; if it yields `len(rings) < 2`, call `chainBoundaryRings` and use its result iff `ok`. The single-edge path stays byte-identical for plain rings.

- [ ] **Step 5: Run the tests to verify they pass** — `go test ./kernel/ops -run TestChainBoundaryRings -v` → PASS (chorded band ≈1144 ±1% watertight; slit pair rejected).

- [ ] **Step 6: Byte-identity + tripwires** — `go test ./kernel/ops -run 'TestRimFilletTorusBand|TestRimFilletWatertight|TestIsFullCircleArc' -v` PASS; `go test ./model/feature -run TestP2TorusBandNotFullDomain -v` PASS (plain path unchanged). Corpus **53** byte-identical. `go build ./kernel/... && go vet ./kernel/ops/ && golangci-lint run ./kernel/ops/...` clean.

- [ ] **Step 7: Commit** — `feat(tessellate): chain a chorded torus-band rim into a ring, gate on congruence`.

---

## Task 3: Unblock the torus setback runout — green T1, faithful T4

**Problem:** with the rim (Task 1) and tessellator (Task 2) fixed, admit torus bosses to the setback path. `setbackBossesFaithful` (`fillet_setback_detect.go`) admits Cylinder/Cone/EllipticalCylinder only; `bodyHasFragileBand` (`fillet_runout_faces.go`) defers torus/b-spline BODIES (a leftover from the OLD split path — the intact path no longer needs it on the runout). Scope the runout deferral so the shared obstacle path stays byte-identical.

**Files:**
- Modify: `kernel/ops/fillet_setback_detect.go` (`setbackBossesFaithful` — admit `geom.Torus`)
- Modify: `kernel/ops/fillet_runout_faces.go` (scope the `bodyHasFragileBand` runout deferral)
- Test: `kernel/ops/fillet_setback_close_test.go` (T1/T4 faithfulness + setback+torus area gate)

- [ ] **Step 1: Failing gate + faithfulness tests** — `go test ./model/feature -run 'TestOCCTBlendSimple/T1$' -v` → FAIL (baseline +1.02%). Add `TestFilletEdges_T1TorusIntact` (T1 result carries the torus wall as a SINGLE un-split face of area ≈**1144.04** ±11, NOT 3947.68; r6 cyl 565.487) and `TestFilletEdges_T4TorusIntact` (torus ≈**2826.04** ±28, NOT 13816.88; r10 cyl 942.478), mirroring `TestFilletEdges_S4WallsIntact`.

- [ ] **Step 2: Verify they fail** — `go test ./kernel/ops -run 'TestFilletEdges_T1TorusIntact|TestFilletEdges_T4TorusIntact' -v` → FAIL (path defers torus / donut area). Expected.

- [ ] **Step 3: Admit torus in the whitelist** — add `geom.Torus` to `setbackBossesFaithful` (keep the per-type structure, one case per proven type).

- [ ] **Step 4: Scope the `bodyHasFragileBand` runout deferral** — `bodyHasFragileBand` is called by BOTH `collectRunouts` and `collectObstacles`. Narrow it on the RUNOUT path only so a torus body is no longer deferred there (the per-boss-type `setbackBossesFaithful` is the real gate now); leave `collectObstacles` untouched. PROVE the obstacle path is byte-identical (diff the obstacle-greened corpus name set — unchanged).

- [ ] **Step 5: GREEN T1 + faithful T4 + oracle gates** — `TestOCCTBlendSimple/T1$` PASS within **[15028.1, 15331.7]**; `TestOCCTBlendSimple/T4$` PASS within **[19319.6, 19709.8]** now through the setback path; `TestFilletEdges_T1TorusIntact`/`T4TorusIntact` PASS (torus single intact face ≈1144/2826). Do NOT loosen a gate; debug against the T1/T4 DRAWEXE oracle if area is off.

- [ ] **Step 6: Setback+torus seam area gate** — add a `kernel/ops` test that fillets a box with a crossing torus boss through the intact setback path and asserts the torus wall face tessellates to ≈1144 (NOT 3947) and the body mesh is watertight (the seam no existing test crosses).

- [ ] **Step 7: Corpus** — simple grid → **54** (S1/S4/T7 + T1, faithful T4), byte-identical elsewhere; broad grid no new failures. `TestP2TorusBandNotFullDomain` still green. Build/vet/lint clean.

- [ ] **Step 8: Commit** — `feat(blend): green T1 + faithful T4 runout (intact torus survivor)`.

---

## Task 4: Milestone verification

**Files:** Modify `model/feature/occtparity/fillet_p2_torus_band_test.go` (T1/T4 remain in the mesh-area set, now producing the correct band).

- [ ] **Step 1: Whole-milestone verification** — full corpus count (simple **54** + broad, no new failures), all tessellation tripwires green, `go build ./kernel/... && go vet && golangci-lint run ./kernel/ops/...` clean. Diff the corpus name set vs base (`477ca07f`) to confirm **only T1/T4 moved**.
- [ ] **Step 2: Confirm T1/T4 are area-gated end-to-end** — `TestP2TorusBandNotFullDomain` (which already lists T1/T4) now asserts the correct band area, not just non-full-domain. If it only checks "not full domain," tighten its T1/T4 entries to the analytic band area (≈1144/2826 ±1%).
- [ ] **Step 3: Commit** — `test(blend): milestone gates for torus setback runout` (only if test changes were needed; otherwise fold into Task 3).

---

## Live test (pre-PR only — this milestone opens NO PR)

Per CLAUDE.md, before the eventual whole-corpus PR: drive `Oblikovati.AddIns.MCPBridge` to fillet a box with a crossing torus boss (a T1-like body), `Recompute`, MCP-screenshot — confirm the torus stands **intact** (no full-donut artefact) and the fillet fades smoothly. This milestone itself opens no PR (the full corpus is not green).

## Notes for the executor

- **Read order per task:** Task 1 → `m4-spike.md` §CRITICAL + `m4-rim-partition-derivation.md` §D2/§D3/§Numerical pitfalls. Task 2 → `m4-spike.md` §(a)(b)(c) + the derivation's congruence gate. Task 3 → the M3 `setbackBossesFaithful`/`bodyHasFragileBand` two-gate composition (ledger `.superpowers/sdd/progress.md`, Task-5 entry).
- Task 1's fix is in **shared** code (S1/S4/T7 use it); byte-identity there is proven by revert-and-diff, not assumed. If the σ-partition changes any cyl/cone sample, stop — the derivation says it should reproduce them exactly.
- No red window: Tasks 1 & 2 land with torus deferred (corpus 53); Task 3 flips T1/T4 green with both fixes already in place.
- Keep the chaining path strictly additive — a plain rim fillet must never enter it (byte-identity gated by `TestP2TorusBandNotFullDomain` + `TestRimFilletTorusBand`).
- Honest-reject over fabrication: if a rim can't partition (full engulfment, >2 crossings) or a rim isn't a congruent 2-ring band, return `ok=false` and keep existing behavior — never force a fill.
