# Corner Extractor Wave — Milestone 3: Unified Intact-Survivor Runout (setback patches)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Re-architect the runout path to OCCT's *true* model — a box-edge fillet running out against **fully-intact** crossing bosses via cylinder-R segments + `G¹` BSpline **setback patches** (host planes re-clipped to single loops). This greens the runout family faithfully by boss surface type: the currently area-coincidental **S1/S4** become topology-faithful and **T7** flips red→green (all kept green throughout).

**Landed status (2026-07-15):** Tasks 1–6 DONE and reviewed clean — corpus **50→53** (S1/S4 now faithful, T7 green), boss-splitting code removed, gate = a per-boss-type `setbackBossesFaithful` whitelist (admits Cylinder/Cone/EllipticalCylinder). **Task 7 (torus T1/T4) is DEFERRED** — blocked on a chorded-rim torus-band tessellator (see Task 7). **S7 (sphere boss)** rides its do-no-harm baseline (green by 0.33%) pending a sphere-cap closure — a follow-up. No PR (the full corpus is not green).

**Architecture:** The RailLoop transfinite-fill engine (`resolveBlend`/coons4, MatchSurface ribbon-`G¹`), the `assembleBody` weld, and the do-no-harm gate are UNCHANGED — this is a re-rail behind the stable seam. The runout tiler stops splitting boss walls; each setback patch is a transfinite fill bounded by the plain-fillet end arc (`G¹` to the fillet cylinder), the **intact** boss footprint conic (`G¹` to the intact boss wall as `Adjacent`), and host-plane seams. New mechanism is built as **unwired** functions first (corpus byte-identical), then swapped in atomically under the per-case oracle gate.

**Tech Stack:** Go (GPL-2.0-only module `oblikovati`), `kernel/ops` B-rep fillet engine; DRAWEXE 8.0.0 oracle.

**Design sources (READ before implementing):**
- `.superpowers/sdd/t1-t7-oracle-forensics.md` — the verified OCCT topology + every measured face area (S1/S4/T1/T4/T7 full inventories, §1–§8).
- `.superpowers/sdd/setback-patch-derivation.md` — the geometry-math-advisor derivation: setback-station formula (D1), 3-patch partition (D2), per-patch rail loop (D3), boss-wall normal laws (D4), pitfalls.

## Global Constraints

- **NO PR until the whole corpus is green.** Accumulate + commit per task on `feat/occt-blend-parity-corpus`.
- **Corpus non-regression, EVERY task:** `go test ./model/feature -run TestOCCTBlendSimple` stays **≥ 52 PASS**, byte-identical on untouched cases. Count: `... -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`. Tasks 2–4 add UNWIRED code → corpus stays exactly 52 byte-identical; only the wiring task (5) and greening tasks (6,7) change the set, and only by ADDING passes.
- **Oracle gate per case (area within OCCT 1 %, `checkprops`):** S1 **3662.79** → [3626.2, 3699.4]; S4 **7004.23** → [6934.2, 7074.3]; T7 **7479.62** → [7404.8, 7554.4]; T1 **15179.9** → [15028.1, 15331.7]; T4 **19514.7** → [19319.6, 19709.8] (must stay green). Never loosen a gate; debug geometry against DRAWEXE.
- **S1/S4 stay green through the swap** — the new path must make them faithful (intact bosses) AND area-passing at the SAME commit the old path is removed. No red window.
- **SPDX** `// SPDX-License-Identifier: GPL-2.0-only` on every new `.go` (`scripts/add-spdx-headers.py`).
- **Style:** funcs 4–20 lines; files < 500; explicit types; early returns, ≤ 2 indent; no duplication; error messages carry offending value + expected shape.
- **Tolerances model-relative** (`ResolutionForBody`/`res.Weld()`, radius-scaled); never a bare `1e-6` in production. Setback interference test: compare `x_s²` to `(k·res.Weld())²`, clamp negative radicand at 0 (derivation D1/pitfalls).
- **Providers stay geom+math only** (no `topo` import in corner providers).
- **DRAWEXE oracle:** `../occt-build/lin64/gcc/bin/DRAWEXE`, env `test-utilities/occt-blend/oracle/drawenv.sh`, `printf 'source X.tcl\n' | DRAWEXE -b`.

---

## Task 1: Split the do-no-harm verdict (obstacle ⟂ runout)

*(Prerequisite from the M2 whole-branch review — independent of the survivor model. A failing runout must never veto a passing obstacle rebuild.)*

**Files:** Modify `kernel/ops/fillet.go` (`assembleFilletBody`), `kernel/ops/fillet_faces.go` (`filletResultFaces`, `collectRebuildFaces`). Test: `kernel/ops/fillet_donoharm_test.go` (new).

**Interfaces:**
- Consumes: `collectObstacles(body,fils,res,maps)` and `collectRunouts(body,fils,res,obHandled,maps)` — identical `(replace,extra,handled)` shape, both short-circuit on `bodyHasFragileBand`.
- Produces: `filletResultFaces(body,fils,blends, enableObstacles,enableRunout bool)`; `assembleFilletBody` signature unchanged.

- [ ] **Step 1: Failing test** — `kernel/ops/fillet_donoharm_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import "testing"

// fakeVerdict scripts pass/fail per candidate composition, standing in for
// obstacleImprovedSolid (which needs a real assembled *topo.Body). Named fake.
type fakeVerdict struct{ pass map[rebuildChoice]bool }

func (f fakeVerdict) improved(c rebuildChoice) bool { return f.pass[c] }

func TestChooseRebuild_RescuesObstacleWhenRunoutOpensShell(t *testing.T) {
	v := fakeVerdict{pass: map[rebuildChoice]bool{
		chooseBoth: false, chooseObstacle: true, chooseRunout: true, chooseBaseline: true}}
	if got := chooseRebuild(v.improved); got != chooseObstacle {
		t.Fatalf("chooseRebuild = %v, want chooseObstacle (obstacle survives a failing {both})", got)
	}
}
func TestChooseRebuild_PrefersBothWhenClean(t *testing.T) {
	v := fakeVerdict{pass: map[rebuildChoice]bool{chooseBoth: true}}
	if got := chooseRebuild(v.improved); got != chooseBoth {
		t.Fatalf("chooseRebuild = %v, want chooseBoth", got)
	}
}
func TestChooseRebuild_FallsToBaseline(t *testing.T) {
	if got := chooseRebuild(fakeVerdict{pass: map[rebuildChoice]bool{}}.improved); got != chooseBaseline {
		t.Fatalf("chooseRebuild = %v, want chooseBaseline", got)
	}
}
```

- [ ] **Step 2: Verify it fails** — `go test ./kernel/ops -run TestChooseRebuild -v` → FAIL (undefined `chooseRebuild`/`rebuildChoice`/constants).

- [ ] **Step 3: Implement the selector** in `kernel/ops/fillet.go`:

```go
type rebuildChoice int

const (
	chooseBoth rebuildChoice = iota // obstacle + runout composed into one watertight solid
	chooseObstacle                  // only the obstacle rebuild improves; runout dropped
	chooseRunout                    // only the runout rebuild improves; obstacle dropped
	chooseBaseline                  // neither improves — the pre-rebuild fillet (do-no-harm)
)

// chooseRebuild picks the highest-priority rebuild composition whose assembled body clears the
// do-no-harm bar. {both} wins when the two rebuilds compose watertight; else the best single path
// (obstacle preferred — the older, more-proven path); else baseline. Splitting the ADR-4 verdict so
// a failing runout can never veto a passing obstacle rebuild (M2 whole-branch review, systemic minor).
func chooseRebuild(improved func(rebuildChoice) bool) rebuildChoice {
	for _, c := range []rebuildChoice{chooseBoth, chooseObstacle, chooseRunout} {
		if improved(c) {
			return c
		}
	}
	return chooseBaseline
}
```

- [ ] **Step 4: Verify it passes** — `go test ./kernel/ops -run TestChooseRebuild -v` → PASS (3/3).

- [ ] **Step 5: Split the enable flag** — in `fillet_faces.go`, `filletResultFaces(..., enableObstacles, enableRunout bool)`; gate `collectObstacles` on `enableObstacles`, `collectRunouts` on `enableRunout` inside `collectRebuildFaces(body,fils,res,maps, enableObstacles,enableRunout)`. Preserve the merge for `{both}`; runout still skips `obHandled`. Keep each func 4–20 lines.

- [ ] **Step 6: Rewrite `assembleFilletBody`** to gate independently:

```go
func assembleFilletBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) *topo.Body {
	cands := rebuildCandidates(body, fils, blends) // lazily assembled; chooseBaseline always present
	choice := chooseRebuild(func(c rebuildChoice) bool {
		b, ok := cands[c]
		return ok && obstacleImprovedSolid(b)
	})
	return cands[choice]
}
```

`rebuildCandidates` assembles baseline (`{false,false}`) always; assembles `{true,true}`/`{true,false}`/`{false,true}` only for enable pairs that actually produce a non-empty `handled` set (so a no-rebuild body stays a single assemble). Keep `obstacleImprovedSolid` (`Valid && IsSolid && HolesContained`).

- [ ] **Step 7: Corpus non-regression + T4 witness** — `go test ./kernel/ops ./model/feature -run 'TestOCCTBlendSimple|TestChooseRebuild|TestCollectRunouts' 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` → **52**; `TestCollectRunouts_DefersOnFragileBand` PASS.

- [ ] **Step 8: Commit** — `refactor(blend): split obstacle+runout do-no-harm into independent verdicts`.

---

## Task 2: Setback stations & band partition (intact-boss detection) — UNWIRED

Build the new detection: for a runout edge, find each crossing boss, keep it intact, compute its setback station `x_s` (contact-line ∩ footprint conic, derivation **D1**), and partition the interfered span into flank/central bands (**D2**). New functions only — NOT called from `runoutFacesFor` yet, so the corpus stays byte-identical.

**Files:** Create `kernel/ops/fillet_setback_detect.go`; Test `kernel/ops/fillet_setback_detect_test.go`. Read `fillet_runout_detect.go` (`runoutImprint`, `detectRunouts`), `corner_runout_region.go` (`detectRunoutRegions`) for the existing boss-crossing detection to reuse.

**Interfaces:**
- Produces:
  ```go
  type crossingBoss struct {
      wall     geom.Surface // the INTACT boss wall (Cylinder/Cone/Torus/SurfaceOfLinearExtrusion)
      footEdge *topo.Edge   // the footprint edge on the host plane (exact conic curve)
      host     *topo.Face   // the support plane the boss exits
      xSetback float64      // signed axial station on the edge where the plain fillet ends (|x_s|)
  }
  type setbackBands struct {
      bosses []crossingBoss // ordered by |xSetback| descending
      cutLo, cutHi float64  // outer setback stations (plain-segment ends), = ±max |x_s|
      seams  []float64      // interior band boundaries (the inner setback stations, ± each smaller x_s)
  }
  func detectSetbackBands(ef edgeFillet, res Resolution) (setbackBands, bool)
  ```
- `xSetback` via `setbackStation(footprint geom.Curve3, contactLine ... , res)` solving line ∩ conic (D1). For a circular footprint radius `r_b`, center distance `a` from edge, `x_s = √(r_b² − (a−R)²)`; general conic → substitute implicit form.

- [ ] **Step 1: Failing test** — pin S1's stations (values from derivation, machine-checked):

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import ("math"; "testing")

// S1: box [-10,10]^3, top-front edge, R=6; r8 top boss (a=10) and r6 front boss (a=10).
func TestDetectSetbackBands_S1TwoBosses(t *testing.T) {
	ef := s1RunoutFixture(t) // named fixture builder (mirror runoutFixtureCrossingBoss)
	b, ok := detectSetbackBands(ef, ResolutionForBody(ef.edge.Body()))
	if !ok || len(b.bosses) != 2 {
		t.Fatalf("detectSetbackBands: ok=%v bosses=%d, want ok=true, 2 crossing bosses (r8,r6)", ok, len(b.bosses))
	}
	// outer = r8: x_s = sqrt(64-16) = sqrt(48); inner = r6: sqrt(36-16) = sqrt(20)
	if math.Abs(b.cutHi-math.Sqrt(48)) > 1e-6 {
		t.Fatalf("outer setback = %v, want sqrt(48)=6.9282 (r8 top boss)", b.cutHi)
	}
	if math.Abs(math.Abs(b.seams[0])-math.Sqrt(20)) > 1e-6 {
		t.Fatalf("inner seam = %v, want sqrt(20)=4.4721 (r6 front boss)", b.seams[0])
	}
}
```

- [ ] **Step 2: Verify it fails** — `go test ./kernel/ops -run TestDetectSetbackBands -v` → FAIL (undefined).

- [ ] **Step 3: Implement** `setbackStation` (line ∩ conic, `x_s²` form, clamp per pitfalls) + `detectSetbackBands` (reuse the boss-crossing detection from `detectRunouts`/`detectRunoutRegions`; keep the boss wall intact — do NOT call any split). Each func 4–20 lines.

- [ ] **Step 4: Verify it passes** + a second test for the `x_s² < (k·res.Weld())²` non-interference clamp (a boss just tangent to the contact line → not a crossing).

- [ ] **Step 5: Corpus byte-identical** — new file unwired: `go test ./model/feature -run TestOCCTBlendSimple 2>&1 | grep -cE '^\s*--- PASS:'` → **52**, name set unchanged.

- [ ] **Step 6: Commit** — `feat(blend): setback-station + band partition for intact-boss runout (unwired)`.

---

## Task 3: Intact-boss setback patch rails (extraction) — UNWIRED

Emit the RailLoops for the flank/central setback patches, bounded by the plain-fillet end arc, the **intact** boss footprint conic (with the intact boss wall as `G¹ Adjacent`), and host-plane seams (derivation **D3**). Reuse the CENTRAL+LEFT+RIGHT structure of the existing `extractRunout`. New extraction path, unwired.

**Files:** Create `kernel/ops/fillet_setback_extract.go`; Test `kernel/ops/fillet_setback_extract_test.go`. Read `corner_extract_runout.go` (current `extractRunout` 3-patch tiler — reuse the loop-chaining helpers `armSectionArc`, seam construction), `corner_rail.go` (`Side`/`RailLoop`).

**Interfaces:**
- Produces `extractSetbackPatches(b setbackBands, ef edgeFillet, res Resolution) ([]RailLoop, bool)` — one RailLoop per band. Each Side: fillet-end quarter-arc `{Curve: arc, Adjacent: ef.cyl, Cont: G1}`; footprint sub-arc `{Curve: footConic, Adjacent: boss.wall, Cont: G1}` (central patch has two); host seam `{Curve: seg, Adjacent: hostPlane, Cont: G1}`; internal band seams `{..., Adjacent: nil, Cont: G0}`.

- [ ] **Step 1: Failing test** — assert each S1 band RailLoop is closed, valence-4, and every non-seam Side carries a non-nil analytic `Adjacent` with `Cont==G1`; assert the footprint Side's `Adjacent` is the intact `geom.Cylinder` (r8 for flanks, r8+r6 for central), NOT a split sub-arc curve.

```go
func TestExtractSetbackPatches_S1IntactFootprintRibbon(t *testing.T) {
	ef := s1RunoutFixture(t)
	b, _ := detectSetbackBands(ef, ResolutionForBody(ef.edge.Body()))
	loops, ok := extractSetbackPatches(b, ef, ResolutionForBody(ef.edge.Body()))
	if !ok || len(loops) != 3 {
		t.Fatalf("extractSetbackPatches: ok=%v loops=%d, want 3 (flank/central/flank)", ok, len(loops))
	}
	for i, lp := range loops {
		if !lp.Closed(ResolutionForBody(ef.edge.Body()).Weld()) {
			t.Fatalf("loop %d not closed", i)
		}
		assertG1FootprintOnIntactWall(t, lp) // no split sub-arc; Adjacent is a full geom.Cylinder
	}
}
```

- [ ] **Step 2: Verify fails.**
- [ ] **Step 3: Implement** `extractSetbackPatches` reusing the 3-patch chaining; the footprint rails are the exact intact conics (`footprintConic` already yields `geom.Circle`/center+radius; extend to hand back the full conic `geom.Curve3`). Flank = fillet-end arc + one footprint arc + 2 host/seam sides; central = fillet-end arc(s) + two footprint arcs + seams (degenerate-3 via `tri3` where a side collapses).
- [ ] **Step 4: Verify passes** + assert each loop `resolveBlend`s to a cert-Valid patch (the engine accepts the rails).
- [ ] **Step 5: Corpus byte-identical (52)** — unwired.
- [ ] **Step 6: Commit** — `feat(blend): intact-boss setback patch rails (unwired)`.

---

## Task 4: Intact-boss closure — host re-clip + no boss split — UNWIRED

Build the watertight closure for the new path: keep every boss wall intact, re-clip each host plane to a **single loop** (open the footprint into the fillet cut — derivation problem-framing / forensics §3), and weld the plain-fillet segments (terminating at `cutLo/cutHi`) + the setback patches to the intact footprint arcs. This replaces `buildSplitBossWall`/notch closure — but is built as new functions, unwired.

**Files:** Create `kernel/ops/fillet_setback_close.go`; Test `kernel/ops/fillet_setback_close_test.go`. Read `fillet_runout_walls.go` (the split closure being replaced), `fillet_runout_hosts.go` (host reconstruction — the re-clip reuses `transformFace`/host-notch machinery but to a single loop), `fillet_runout_faces.go` (`runoutWings`, `sampledArcSegs` — the segments now end at `x_s`).

**Interfaces:**
- Produces `buildSetbackFaces(set *runoutSet, ef edgeFillet, b setbackBands, loops []RailLoop, res, maps) bool` — appends: (a) 2 plain cyl-R segment faces (wings ending at `cutLo/cutHi`); (b) the resolved setback patch faces; (c) the re-clipped single-loop host planes (footprint arc becomes part of the outer loop, sampled identically to the patch footprint rails for the weld); it does NOT emit any boss-wall face (the intact wall stays via `transformedBodyFaces`, unreplaced).

- [ ] **Step 1: Failing test** — build S1 faces via `buildSetbackFaces`, assemble, assert: watertight solid (`IsSolid`), every boss wall face present & area-intact (r8 753.982, r6 565.487 — byte-preserved), every host plane single-loop (`WIRE:1`), and total area in S1's window [3626.2, 3699.4].
- [ ] **Step 2: Verify fails.**
- [ ] **Step 3: Implement** re-clip (single-loop host: open footprint into the cut, no inner ring) + segment termination at `x_s` + weld sampling (`ringSegSamples`, matched to patch rails). Do NOT touch the intact boss wall.
- [ ] **Step 4: Verify passes** — S1 watertight + area faithful + bosses intact.
- [ ] **Step 5: Corpus byte-identical (52)** — unwired.
- [ ] **Step 6: Commit** — `feat(blend): intact-boss host re-clip + setback closure (unwired)`.

---

## Task 5: Wire the intact-boss path; make S1/S4 faithful; remove split code

Atomic swap: `runoutFacesFor` calls the new `detectSetbackBands → extractSetbackPatches → buildSetbackFaces` path instead of the old `detectRunoutRegions → extractRunoutTiled → buildRunoutHostsAndWalls` split path. Gate on the full corpus: S1/S4 stay green **and** are now topology-faithful (intact bosses). Remove the now-dead split code (`buildSplitBossWall`, `bossWallSubArcs`, the old `extractRunoutTiled` hexagon tiler) once green.

**Files:** Modify `kernel/ops/fillet_runout_faces.go` (`runoutFacesFor`/`appendRegionFaces` → call new path); delete/retire `fillet_runout_walls.go` split helpers + the old region tiler paths made dead. Also revisit `bodyHasFragileBand`: under the intact model a torus/oblique-ellipse boss is a legal survivor, so the short-circuit must NOT block a valid setback build — refine it to "defer only when no valid setback patch resolves," preserving T4 (Task 7 tightens this; here keep S1/S4/non-torus correct and leave torus bodies deferring to baseline as today so T4 stays green).

- [ ] **Step 1: Failing test / gate** — `go test ./model/feature -run 'TestOCCTBlendSimple/(S1|S4)$' -v` currently PASS via split; after the swap they must PASS via the intact path. Add `TestSetback_S1BossesIntact` / `TestSetback_S4ConeIntact` asserting the RESULT body's boss walls are byte-area-preserved (proves faithful, not coincidental): S1 r6 565.487 + r8 753.982; S4 cone 1218.1 + cyl-r10 942.478.
- [ ] **Step 2: Wire** the new path in `runoutFacesFor`; keep the `bodyHasFragileBand` deferral for torus/bspline bodies (T1/T4/T7 still defer here — greened in Tasks 6–7).
- [ ] **Step 3: Oracle-gate S1/S4** — `go test ./model/feature -run 'TestOCCTBlendSimple/(S1|S4)$' -v` → PASS within window; `TestSetback_*Intact` PASS.
- [ ] **Step 4: Remove dead split code** — delete `buildSplitBossWall`, `bossWallSubArcs`, `traceRimSubArcs` (if now unused), the old hexagon `extractRunout`/`extractRunoutTiled` + `buildRunoutHostsAndWalls`. `go build ./kernel/... && go vet ./kernel/ops/` clean.
- [ ] **Step 5: Corpus non-regression** — count **52**, byte-identical name set (S1/S4 still green via new path; T4 still green at baseline; nothing else moved). `TestCollectRunouts_DefersOnFragileBand` PASS.
- [ ] **Step 6: Commit** — `refactor(blend): S1/S4 runout now topology-faithful (intact bosses); drop boss-splitting`.

---

## Task 6: T7 — oblique elliptical-cylinder survivor

Extend the footprint/boss handling to the oblique elliptical cylinder (`SurfaceOfLinearExtrusion` of an ellipse) so T7 greens. Footprint on the host plane is an **ellipse** (`geom.Ellipse`, rational quadratic); the boss-wall normal is `C′(u) × v` (derivation D4). No boss split.

**Files:** Modify `kernel/ops/fillet_setback_detect.go` (`setbackStation` for an elliptical footprint — line ∩ ellipse), `fillet_setback_extract.go` (footprint rail = `geom.Ellipse` sub-arc; `Adjacent` = the `SurfaceOfLinearExtrusion`). Read `geom` for the exact ellipse & surface-of-extrusion normal. Refine `bodyHasFragileBand` so an oblique-ellipse survivor no longer forces baseline when a valid setback resolves.

- [ ] **Step 1: Failing gate** — `go test ./model/feature -run 'TestOCCTBlendSimple/T7$' -v` → FAIL at baseline surplus. Add `TestSetback_T7EllipseSurvivorIntact` asserting the result carries the intact oblique-ellipse wall (area 2381.68) + r8 cyl (603.186).
- [ ] **Step 2: Implement** the elliptical footprint station + rail + `Adjacent` normal path (verify `MatchSurface` reads the `SurfaceOfLinearExtrusion` normal; if not, supply the analytic `C′(u)×v` normal to the ribbon).
- [ ] **Step 3: Oracle-gate T7** — PASS within [7404.8, 7554.4]; survivor areas intact.
- [ ] **Step 4: Corpus** — **53**, byte-identical elsewhere; T4 still green.
- [ ] **Step 5: Commit** — `feat(blend): green T7 runout (oblique elliptical-cylinder survivor)`.

---

## Task 7: Torus footprint spike + T1/T4 — torus survivors — ⛔ DEFERRED (2026-07-15)

**Outcome: BLOCKED at the Step-0 spike; deferred to a torus-band tessellator follow-up (user decision 2026-07-15).**
The spike proved the runout *logic* is ready (the T1/T4 torus↔host footprint is a circle, DRAWEXE-confirmed; only the whitelist + fragile-band scoping remained) but hit a **tessellation-infrastructure** gap: keeping the torus a single intact face, the runout weld still chord-subdivides its footprint rim into ~26 `geom.LineSegment` sub-edges, and `bandRingsAndSeam` (`closed_band_loft.go:70`) recognizes a ring ONLY as a single `geom.Circle`/full-sweep `Arc3d` → `closedBandLoftMesh` fails → `fullDomainGridMesh` grids the whole `[0,2π]²` domain → the intact torus meshes as the FULL DONUT (T1 3947.68 vs OCCT 1144.04; T4 13816.9 vs 2826.04) — exactly the `tessellate_trim.go:22` gap. Per CLAUDE.md, tessellation correctness is the highest priority; the fix (a chorded-rim torus-band mesher — coalesce coplanar-circle sub-edges into a ring, or a periodic-band CDT) is its own spike→plan→SDD sub-project and very likely also unblocks T9/S9/T3. See `.superpowers/sdd/task-7-report.md`. Steps below stand as the recipe for when the tessellator lands.

Green T1 and make T4 faithful. **First a verification spike** (advisor flag D4): confirm via DRAWEXE whether the T1/T4 torus footprint on the host plane is a conic (circle) or a higher-degree curve — this decides the rail type. Then extract the torus footprint rail + intact torus wall as `G¹ Adjacent` (no split — sidesteps the split-rim torus-band tessellation gap `tessellate_trim.go:22`).

**Files:** Modify `kernel/ops/fillet_setback_detect.go`/`fillet_setback_extract.go` (torus footprint rail + `geom.Torus` `Adjacent`); tighten `bodyHasFragileBand` so a torus survivor with a resolvable setback proceeds while a genuinely un-buildable body still defers (keep T4 green either way — if T4's setback resolves it's faithful; if not it defers to its green baseline).

- [ ] **Step 0: Spike** — run DRAWEXE on T1/T4: dump the torus↔host-plane footprint edge geometry (`whatis`/`dump` the edge's curve). Record in a comment + the ledger whether it's a `Circle`/`Ellipse`/`BSpline`. If non-conic, the rail is the exact `Torus ∩ Plane` NURBS (do not approximate).
- [ ] **Step 1: Failing gate** — `TestOCCTBlendSimple/T1$` → FAIL (+1.02% baseline). Add `TestSetback_T1TorusIntact` (torus wall 1144.04 intact, NOT the 3947.84 full donut) and re-assert `TestSetback_T4TorusIntact` (2826.04).
- [ ] **Step 2: Implement** the torus footprint rail (per spike) + intact-torus `Adjacent` ribbon; ensure the intact torus wall is never split (`buildSplitBossWall` is already gone after Task 5).
- [ ] **Step 3: Oracle-gate T1** — PASS within [15028.1, 15331.7]; torus intact (assert result torus face area 1144.04, single face).
- [ ] **Step 4: T4 faithful & green** — `TestOCCTBlendSimple/T4$` PASS within [19319.6, 19709.8]; assert its torus wall (2826.04) + cyl-r10 (942.478) intact — now green for the RIGHT reason, not tolerance luck.
- [ ] **Step 5: Corpus** — **54** (S1,S4,T7,T1 green + T4 faithful); byte-identical elsewhere. Full run: `go test ./kernel/ops ./model/feature -run 'TestOCCTBlend|TestSetback|TestChooseRebuild|TestCollectRunouts'` all green.
- [ ] **Step 6: Commit** — `feat(blend): green T1 + faithful T4 runout (intact torus survivors)`.

---

## Live test (before any PR — corpus not yet fully green, so still NO PR)

Per CLAUDE.md, once the corpus is green enough to warrant it: drive `Oblikovati.AddIns.MCPBridge` to fillet a box with a crossing cylinder + torus boss (an S1/T1-like body), `Recompute`, and MCP-screenshot the result — confirm the boss stands intact and the fillet fades out smoothly with no split-rim artefact. (This slice does not itself open a PR; the branch keeps accumulating until the whole `tests/blend` corpus is green.)

---

## Notes for the executor

- Tasks 2–4 are deliberately UNWIRED so the corpus is provably byte-identical (52) at each — the risk concentrates in the single atomic swap (Task 5), gated by S1/S4 staying green via the new path.
- The engine (`resolveBlend`, coons4/tri3, MatchSurface, `assembleBody`, do-no-harm) is not modified beyond Task 1 — if a patch won't resolve, honest-reject the edge (do-no-harm to baseline), never a partial fill.
- Every "faithful" claim is gated by a boss-wall-area-intact assertion (not just total area), so a future area-coincidence cannot masquerade as correctness again.
