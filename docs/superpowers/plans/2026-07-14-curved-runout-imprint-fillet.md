# Curved-Runout Imprint Fillet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Green the 5 remaining thread-③ `FAIL(area)` corpus cases (simple S1, S4, T1, T7, T9) — a constant-radius straight-edge (plane∧plane) cylinder fillet whose runout footprint crosses a pre-existing curved feature's base curve on the shared host plane, causing a stable +1.15–1.63% area surplus — by imprinting the feature footprint into the receded fillet band and trimming the fillet where the feature covers it.

**Architecture:** A NEW, lightweight "runout imprint" path in `kernel/ops`, SEPARATE from the mid-span obstacle rebuild (ADR-4). Where the obstacle path notches the host + splits the tube wall + adds wings + a FillSurface patch, the runout path only **(a)** merges the coplanar feature footprint into the receded band on each host plane and **(b)** trims the fillet strip the feature covers — emitting **no** new blend faces (Step-0 oracle confirmed OCCT keeps the curved walls/caps byte-identical). It reuses the obstacle detector's host-plane primitives; the only per-family code is the footprint∩line imprint solve (closed-form conic for S1/S4/T1/T7, marched for T9). It wires into `assembleFilletBody` behind the existing do-no-harm "area-improved solid" fallback, so a mis-fire can never regress the green corpus.

**Tech Stack:** Go (GPL module `oblikovati`), `kernel/ops` + `kernel/geom` + `kernel/topo`; DRAWEXE 8.0.0 parity oracle; `model/feature` corpus harness (`TestOCCTBlendSimple`).

## Global Constraints

- Branch `feat/occt-blend-parity-corpus`. **NO PR until the whole corpus is green** — accumulate and commit per task on the branch.
- SPDX `GPL-2.0-only` header on every new `.go` file (`scripts/add-spdx-headers.py`).
- Functions 4–20 lines; files < 500 lines; explicit types (no `any`/untyped); early returns; max 2 indent levels; no code duplication.
- TDD: every new function gets a test; external I/O mocked with **named fakes**, never inline stubs.
- **Never regress the green corpus.** `TestOCCTBlendSimple` currently = **50 PASS / 145 FAIL**. After each task PASS must be ≥ its prior value, and every case not targeted by the task must be **byte-identical** (same PASS/FAIL, same area) — verified by a full per-case before/after diff.
- Tolerances scale to the body's bounding diagonal (existing `importTol = 1e-4·diag` pattern; `res.Weld()` for model-relative weld).
- Fold of `HolesContained` into `Validate.Valid` stays DEFERRED (tripwire only) — out of scope here.

## Verified ground truth (Step-0 DRAWEXE reconciliation, S1)

- OCCT S1 = 15 faces, total area **3662.79** (== corpus reference). Ours = 11 faces, **3712.64**. Surplus **+49.85**.
- BYTE-IDENTICAL OCCT-vs-ours (DO NOT touch these): r8 boss wall 753.982 (=2π·8·15), r6 boss wall 565.487 (=2π·6·15), r8 top cap 201.062 (=π·8²), r6 end cap 113.097 (=π·6²), four box faces 400, 400, 392.274, 392.274. Σ matched = 3218.18.
- The +49.85 is entirely in the **fillet + adjacent receded planes**. Our fillet is the **full untrimmed quarter-cylinder 188.495** (=½π·6·20). S1 has TWO bosses on TWO host planes (r8 vertical on top plane z=10; r6 horizontal on front plane y=−10), each crossing the receded band.
- The exact split of the surplus between "trim the fillet" and "merge the plane lens" is NOT yet localized (Step-0 matched 8 faces by area; the other 7 OCCT faces were inferred). **Task 1 localizes it definitively via a per-face surface-type dump.** Everything downstream is the confirmed Mode-A architecture (surplus in the fillet+plane region; walls/caps untouched; fix = imprint footprint → merge band + trim fillet).

## File Structure

- `kernel/ops/fillet_runout_detect.go` (NEW) — runout-imprint detection: find, per host plane, a coplanar feature footprint whose base curve crosses the receded fillet band. Reuses `boundaryLine2`, `rimCrossings`, `obstacleNodes`, `dipsPast`, `planeFrame`, `boundaryFromTangents`, `hostTangents`, `singleHoleEdge`, `sampleHoleRim` from `fillet_obstacle_detect*.go`.
- `kernel/ops/fillet_runout_imprint.go` (NEW) — the tiered footprint∩line solver (closed-form conic; marched for b-spline) returning the two exact crossing nodes P± and the footprint sub-arc between them.
- `kernel/ops/fillet_runout_apply.go` (NEW) — apply one imprint: merge the footprint into the host-plane band loop (reusing the `8d8c184f` crossing-loop merge) and trim the fillet strip; return replaced/updated `filletFace`s.
- `kernel/ops/fillet.go` (MODIFY) — `assembleFilletBody` gains the runout path behind the do-no-harm `obstacleImprovedSolid`-style fallback.
- `kernel/ops/fillet_faces.go` (MODIFY, only if the fillet trim needs a hook in `filletResultFaces`) — thread the runout imprint set through, mirroring `enableObstacles`.
- `kernel/ops/fillet_runout_*_test.go` (NEW) — unit tests per new function with a named `fakeFilletHost`/synthetic bodies.
- `model/feature/occtparity/fillet_runout_test.go` (NEW) — per-family corpus gate (S1, S4, T1, T7, T9) asserting `FAIL(area)→PASS` + zero regression.
- `test-utilities/occt-blend/oracle/facetypes.tcl` (NEW) — the per-face surface-type + area DRAWEXE dump (Task 1).
- `docs/superpowers/specs/2026-07-13-curved-corner-miter-blend-engine-design.md` (MODIFY) — add **ADR-5** recording the runout-imprint decision (merge+trim, no new faces, separate from ADR-4 obstacle rebuild).

---

### Task 1: Oracle localization — per-face surface-type dump (diagnostic gate)

Definitively localize the +49.85 surplus (fillet vs plane) and read OCCT's exact imprint. This unblocks the trim-vs-merge emphasis in Tasks 4–5. No production code.

**Files:**
- Create: `test-utilities/occt-blend/oracle/facetypes.tcl`

**Interfaces:**
- Produces: a documented mapping (added to this plan's Task-5 notes and to ADR-5) of each OCCT S1 result face → {surface type, area}, and which faces are the trimmed fillet vs the merged planes.

- [ ] **Step 1: Write the dump script**

`test-utilities/occt-blend/oracle/facetypes.tcl`:
```tcl
# SPDX-License-Identifier: GPL-2.0-only
# Reproduce an OCCT blend on a corpus STEP input and dump per-face surface type + area.
# Usage: STEP=<abs path> EDGEMID={x y z} RAD=<r> ; printf 'source facetypes.tcl\n' | DRAWEXE -b
pload MODELING
pload STEP
set stp $env(STEP)
set rad $env(RAD)
set mid $env(EDGEMID)
stepread $stp a *
explode a_1 E
set target ""
foreach e [directory a_1_*] {
    if {[catch { mkcurve __c $e }]} { continue }
    bounds __c __lo __hi
    set um [expr {([dval __lo]+[dval __hi])/2.0}]
    cvalue __c $um px py pz dx dy dz
    set dd [expr {abs([dval px]-[lindex $mid 0])+abs([dval py]-[lindex $mid 1])+abs([dval pz]-[lindex $mid 2])}]
    if {$dd < 0.6} { set target $e }
}
puts "TARGET=$target"
blend result a_1 $rad $target
explode result F
foreach f [directory result_*] {
    set typ [whatis $f]
    set pr  [sprops $f]
    puts "FACEDUMP $f | type=$typ | $pr"
}
puts "DONE"
exit
```

- [ ] **Step 2: Run it for S1 and record the face types**

Run:
```bash
cd /home/vmiguel/git/oblikovati-workspace/Oblikovati
source test-utilities/occt-blend/oracle/drawenv.sh
STEP="$PWD/model/feature/occtparity/fixtures/simple/S1.step" RAD=6 EDGEMID="0 -10 10" \
  printf 'source test-utilities/occt-blend/oracle/facetypes.tcl\n' | timeout -s KILL 90 "$DRAWEXE" -b 2>&1 \
  | grep -E 'FACEDUMP|Mass|DONE'
```
Expected: 15 `FACEDUMP` lines. Use `dumpsurface`/`whatis`/`nbshapes` as needed to classify each: identify the fillet faces (cylinder radius 6, axis ∥ x) and sum their area, and the receded plane faces and sum theirs. Record the two sums.

- [ ] **Step 3: Localize and record the mechanism**

Compare to ours (fillet 188.495; receded planes = the two `loops=2` planes 122.171 + 183.802 = 305.97). Record in this plan and ADR-5:
- If OCCT fillet sum ≈ 145 (ours 188.5, Δ≈+43) → **fillet-trim dominant**; Tasks 4 (trim) carries most of the fix.
- If OCCT plane sum ≪ 305.97 (Δ≈+43) and OCCT fillet ≈ 188 → **plane-merge dominant**; Task 5 (merge) carries most of the fix.
- Record the exact imprint edge OCCT puts on the fillet (its curve type via `whatis` on the fillet face's boundary edges) — this is the trim curve Task 4 must reproduce.

- [ ] **Step 4: Commit**

```bash
git add test-utilities/occt-blend/oracle/facetypes.tcl docs/superpowers/plans/2026-07-14-curved-runout-imprint-fillet.md
git commit -m "diag(fillet): per-face surface-type oracle dump; localize S1 runout surplus (fillet vs plane)"
```

---

### Task 2: Runout imprint detection (per host plane, coplanar footprint crossing the band)

Detect, for each of the fillet's two planar faces, a coplanar feature footprint whose base curve crosses the receded fillet band twice and dips into it — the runout-imprint trigger. This is where S1's two independent bosses (which `detectObstacle` rejects as `qualifying==2`) are each admitted independently.

**Files:**
- Create: `kernel/ops/fillet_runout_detect.go`
- Test: `kernel/ops/fillet_runout_detect_test.go`

**Interfaces:**
- Consumes: `boundaryLine2`, `crossing`, `rimCrossings`, `obstacleNodes`, `dipsPast` (`fillet_obstacle_detect.go`); `planeFrame`, `hostTangents`, `boundaryFromTangents`, `filletBandSide`, `singleHoleEdge`, `sampleHoleRim` (`fillet_obstacle_detect_face.go`); `edgeFillet`, `Resolution`.
- Produces:
  ```go
  type runoutImprint struct {
      host          *topo.Face          // the fillet's planar face carrying the footprint
      hostIsA       bool
      plane         geom.Plane
      footprintEdge *topo.Edge          // the closed feature base curve on host (inner loop)
      nodes         [2]crossing         // the two band crossings, host-plane 2D
      flat          func(math.Point3) math.Point2
      back          func(math.Point2) math.Point3
  }
  func detectRunouts(ef edgeFillet, res Resolution) []runoutImprint
  ```
  Returns one entry per (host, footprint) that crosses+dips. Empty when none (leaves the fillet whole — the benign S3/S6/S7/T3 non-crossing siblings).

- [ ] **Step 1: Write the failing test** (`fillet_runout_detect_test.go`)

Build a synthetic slab (plane∧plane straight edge) with a coplanar circular boss footprint that crosses the receded band on ONE face, and assert exactly one imprint with two nodes; plus a NON-crossing boss (footprint entirely behind the band) asserting zero imprints. Use a named `runoutFixture` helper that assembles the `edgeFillet` + faces (mirror the setup in `fillet_obstacle_watertight_test.go`).
```go
func TestDetectRunouts_SingleCrossingBoss(t *testing.T) {
    ef, res := runoutFixtureCrossingBoss(t) // slab + coplanar boss dipping into the band
    got := detectRunouts(ef, res)
    if len(got) != 1 { t.Fatalf("want 1 imprint, got %d", len(got)) }
    if got[0].nodes[0].P == got[0].nodes[1].P { t.Fatalf("want two distinct nodes, got coincident: %+v", got[0].nodes) }
}
func TestDetectRunouts_NonCrossingBoss(t *testing.T) {
    ef, res := runoutFixtureBehindBand(t) // boss entirely on host side of the band
    if got := detectRunouts(ef, res); len(got) != 0 {
        t.Fatalf("non-crossing boss must produce no imprint, got %d", len(got))
    }
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run TestDetectRunouts -count=1 -v`
Expected: FAIL — `detectRunouts` undefined.

- [ ] **Step 3: Implement `detectRunouts`**

Only fires on a straight plane∧plane fillet edge. For each host, reuse the obstacle detector's per-host plumbing but WITHOUT the `qualifying==1` / `rebuildableTube` / wall-plane gates (those are obstacle-specific). Keep functions ≤20 lines by splitting a `runoutOnHost` helper.
```go
func detectRunouts(ef edgeFillet, res Resolution) []runoutImprint {
    if ef.varying || !straightFilletEdge(ef, res) {
        return nil
    }
    var out []runoutImprint
    for _, hostIsA := range []bool{true, false} {
        host := ef.b
        if hostIsA {
            host = ef.a
        }
        if im, ok := runoutOnHost(ef, host, hostIsA, res); ok {
            out = append(out, im)
        }
    }
    return out
}

func runoutOnHost(ef edgeFillet, host *topo.Face, hostIsA bool, res Resolution) (runoutImprint, bool) {
    pl, ok := host.Geometry().(geom.Plane)
    if !ok {
        return runoutImprint{}, false
    }
    fp, ok := singleHoleEdge(host)
    if !ok || fp.StartVertex() != fp.EndVertex() {
        return runoutImprint{}, false
    }
    flat, back := planeFrame(pl)
    b, ok := boundaryFromTangents(ef, hostIsA, flat)
    if !ok {
        return runoutImprint{}, false
    }
    rim := sampleHoleRim(fp.Geometry(), fp.ID())
    rim2 := project2(rim, flat) // []math.Point2
    nodes, ok := obstacleNodes(rim2, b, res)
    if !ok {
        return runoutImprint{}, false
    }
    side := filletBandSide(ef, b, flat)
    if !dipsPast(rim2, nodes[0], nodes[1], b, side) {
        return runoutImprint{}, false
    }
    return runoutImprint{host, hostIsA, pl, fp, nodes, flat, back}, true
}
```
Add a small `project2(loop filletLoop, flat func(math.Point3) math.Point2) []math.Point2` helper (or reuse an existing projector if `sampleHoleRim` already returns 2D — check its return type and adapt; do not duplicate a projector that exists).

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./kernel/ops/ -run TestDetectRunouts -count=1 -v`
Expected: PASS (both subtests).

- [ ] **Step 5: Add SPDX + commit**

```bash
python3 scripts/add-spdx-headers.py kernel/ops/fillet_runout_detect.go kernel/ops/fillet_runout_detect_test.go
git add kernel/ops/fillet_runout_detect.go kernel/ops/fillet_runout_detect_test.go
git commit -m "feat(fillet): detect coplanar runout-imprint crossings per host plane"
```

---

### Task 3: Tiered footprint∩line imprint solve (exact conic first)

Given a `runoutImprint`, compute the EXACT crossing points P± and the footprint sub-arc between them, for the closed-form conic families (circle now; ellipse in Task 8). The sampled `nodes` from Task 2 are polyline-approximate; area parity to <1% wants the exact conic crossing and the exact sub-arc.

**Files:**
- Create: `kernel/ops/fillet_runout_imprint.go`
- Test: `kernel/ops/fillet_runout_imprint_test.go`

**Interfaces:**
- Consumes: `runoutImprint`; `geom.Circle`/footprint geometry; `res Resolution`.
- Produces:
  ```go
  type imprintCut struct {
      pMinus, pPlus math.Point3 // exact crossing points (3D, on the host plane)
      arc           geom.Curve3 // footprint sub-arc between them, on the OUTBOARD side (away from the band)
  }
  func solveImprint(im runoutImprint, res Resolution) (imprintCut, bool) // false ⇒ tangential/grazing, do not imprint
  ```

- [ ] **Step 1: Write the failing test**

For a unit circle footprint (center (0,0), r=8) and a band line y=−4 in the host-plane frame, assert P± = (±√48, −4) within `1e-9`, and that a tangential line (y=−8, tangent to r=8) returns `ok=false`.
```go
func TestSolveImprint_CircleCrossing(t *testing.T) {
    im := runoutImprintCircle(math.P2(0, 0), 8, lineY(-4)) // named fixture
    cut, ok := solveImprint(im, unitRes())
    if !ok { t.Fatal("want crossing, got tangential") }
    wantX := math.Sqrt(48)
    if math.Abs(math.Abs(cut.pMinus.X)-wantX) > 1e-9 { t.Fatalf("P- x=%v want ±%v", cut.pMinus.X, wantX) }
}
func TestSolveImprint_TangentRejected(t *testing.T) {
    im := runoutImprintCircle(math.P2(0, 0), 8, lineY(-8))
    if _, ok := solveImprint(im, unitRes()); ok { t.Fatal("tangential line must not imprint") }
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run TestSolveImprint -count=1 -v`
Expected: FAIL — `solveImprint` undefined.

- [ ] **Step 3: Implement the circle∩line closed form + sub-arc**

In the host-plane 2D frame, the band is `boundaryLine2{origin,dir}` (dir unit). For a circle center C radius r: substitute P(t)=origin+t·dir into |P−C|²=r²:
```go
// t² + 2t·(dir·(origin−C)) + (|origin−C|² − r²) = 0
func lineCircleRoots(b boundaryLine2, c math.Point2, r, scale float64) (t0, t1 float64, ok bool) {
    w := c.VectorTo(b.origin) // origin − C
    bb := b.dir.Dot(w)
    cc := w.Dot(w) - r*r
    disc := bb*bb - cc
    if disc < (scale*imprintGrazeEps)*(scale*imprintGrazeEps) { // tangential/grazing guard
        return 0, 0, false
    }
    s := math.Sqrt(disc)
    return -bb - s, -bb + s, true
}
```
`scale` = the host's bounding diagonal (model-relative, ADR-0042); `imprintGrazeEps` a small dimensionless constant (e.g. `1e-6`) documented as the grazing threshold. Map the two roots back to 3D via `im.back`, and extract the footprint sub-arc on the outboard side (the major arc, away from the band) using the exact circle parameterization (`geom` circle `PointAt`/knot-insert; do not re-sample). Keep each function ≤20 lines.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./kernel/ops/ -run TestSolveImprint -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Add SPDX + commit**

```bash
python3 scripts/add-spdx-headers.py kernel/ops/fillet_runout_imprint.go kernel/ops/fillet_runout_imprint_test.go
git add kernel/ops/fillet_runout_imprint*.go
git commit -m "feat(fillet): exact circle∩band imprint solve with grazing guard"
```

---

### Task 4: Apply imprint — merge footprint into band + trim fillet

Turn one `imprintCut` into modified faces: the host plane's band loop merged with the footprint (lens removed) and the fillet trimmed at the imprint. Emit NO new blend faces.

**Files:**
- Create: `kernel/ops/fillet_runout_apply.go`
- Test: `kernel/ops/fillet_runout_apply_test.go`

**Interfaces:**
- Consumes: `runoutImprint`, `imprintCut`, the `filletRebuildMaps`/`filletFace` machinery; the crossing-loop merge introduced in commit `8d8c184f` (locate it in `fillet_obstacle_faces.go` / `buildNotchedHost`; REUSE, do not re-implement the splice).
- Produces:
  ```go
  type runoutResult struct {
      replace map[uint64]filletFace // host-face ID → merged planar face; fillet-face key → trimmed fillet
  }
  func applyImprints(ef edgeFillet, ims []runoutImprint, cuts []imprintCut, maps filletRebuildMaps) (runoutResult, bool)
  ```

- [ ] **Step 1: Write the failing test**

On the synthetic slab+crossing-boss fixture, apply the imprint and assert (a) the merged host plane area equals `squareArea − bandStrip − footprintOutsideBand` (the lens is NOT double-counted), computed analytically in the test, and (b) the trimmed fillet area is strictly less than the full quarter-cylinder by the expected imprint amount. Compute both expected values by hand in the test from the fixture dimensions.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run TestApplyImprints -count=1 -v`
Expected: FAIL — `applyImprints` undefined.

- [ ] **Step 3: Implement merge + trim**

Localization from Task 1 decides the emphasis; implement BOTH, each guarded so a zero-width result is a no-op:
- **Merge:** split the host plane's outer band loop and the footprint inner loop at P±; reuse the `8d8c184f` splice to form ONE merged outer loop excluding the lens. Produce the replacement `filletFace` for `im.host.ID()`.
- **Trim:** split the fillet's boundary rail at the axial positions of P± and re-route it along the imprint curve OCCT uses (from Task 1's recorded imprint-edge type); trim the covered strip. Produce the replacement fillet `filletFace`.
Keep helpers ≤20 lines; factor `mergeHostBand`, `trimFilletStrip`.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./kernel/ops/ -run TestApplyImprints -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Add SPDX + commit**

```bash
python3 scripts/add-spdx-headers.py kernel/ops/fillet_runout_apply.go kernel/ops/fillet_runout_apply_test.go
git add kernel/ops/fillet_runout_apply*.go
git commit -m "feat(fillet): apply runout imprint — merge band + trim fillet (no new faces)"
```

---

### Task 5: Wire the runout path into assembleFilletBody with do-no-harm fallback

Integrate detection → solve → apply into the fillet build, guarded so a mis-fire can never regress the corpus.

**Files:**
- Modify: `kernel/ops/fillet.go` (`assembleFilletBody`)
- Modify: `kernel/ops/fillet_faces.go` (thread the runout replacements through `filletResultFaces`, mirroring `enableObstacles`) — only if the trim needs a build hook.
- Test: `kernel/ops/fillet_runout_watertight_test.go`

**Interfaces:**
- Consumes: `detectRunouts`, `solveImprint`, `applyImprints`; `obstacleImprovedSolid` / `Validate` / `IsSolid`.
- Produces: `assembleFilletBody` returns the runout-imprinted body when it is a valid solid whose area moved toward parity; else the baseline body (unchanged).

- [ ] **Step 1: Write the failing test**

Synthetic slab + coplanar boss whose footprint crosses the band: assert the assembled body `IsSolid()`, `Validate().Valid`, and total area equals the analytic hand-computed area (surplus removed).

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./kernel/ops/ -run TestFilletRunoutWatertight -count=1 -v`
Expected: FAIL.

- [ ] **Step 3: Implement the wiring + fallback**

```go
func assembleFilletBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) *topo.Body {
    base := buildFilletBody(body, fils, blends) // existing path (extract if needed)
    ims := collectRunoutImprints(body, fils)     // detect+solve+apply across all fillets
    if len(ims) == 0 {
        return base
    }
    imprinted := buildFilletBodyWithRunout(body, fils, blends, ims)
    if runoutImproved(base, imprinted) {
        return imprinted
    }
    return base
}

// runoutImproved keeps the imprinted body only if it is a valid solid whose surface area
// DECREASED vs the baseline (the surplus is a positive area error, so a correct imprint can
// only shrink area). Any mis-fire (larger/equal area, or invalid) falls back — corpus-safe.
func runoutImproved(base, imp *topo.Body) bool {
    r := Validate(imp)
    if !r.Valid || !imp.IsSolid() {
        return false
    }
    a := BodyGeometryProperties(imp, PropertyQuality()).Area
    b := BodyGeometryProperties(base, PropertyQuality()).Area
    return a < b-areaImproveEps
}
```
`areaImproveEps` = a model-relative epsilon (scaled to `base` area) so noise-level changes do not count as improvement. Keep each function ≤20 lines.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./kernel/ops/ -run TestFilletRunout -count=1 -v && go test ./kernel/ops/ -count=1`
Expected: PASS; whole `ops` suite green.

- [ ] **Step 5: Commit**

```bash
python3 scripts/add-spdx-headers.py kernel/ops/fillet_runout_watertight_test.go
git add kernel/ops/fillet.go kernel/ops/fillet_faces.go kernel/ops/fillet_runout_watertight_test.go
git commit -m "feat(fillet): wire runout imprint into assembleFilletBody with area-improved fallback"
```

---

### Task 6: Gate S1 green + zero-regression proof

Prove the end-to-end path greens S1 and regresses nothing.

**Files:**
- Create: `model/feature/occtparity/fillet_runout_test.go`

**Interfaces:**
- Consumes: `RunCase`, `Corpus`, `CorpusFixtureDir` (occtparity package).
- Produces: a per-family gate test.

- [ ] **Step 1: Write the gate test**

```go
func TestRunoutFamilyPasses(t *testing.T) {
    for _, id := range []string{"S1"} { // grows to S4,T1,T7,T9 as tasks land
        r := caseByID(t, "simple", id) // named lookup helper over Corpus()
        RunCase(t, r, CorpusFixtureDir())
    }
}
```

- [ ] **Step 2: Run the targeted case**

Run: `go test ./model/feature/ -run 'TestOCCTBlendSimple/S1' -count=1 -v`
Expected: PASS (was FAIL(area) +1.36%).

- [ ] **Step 3: Full corpus zero-regression diff**

Run before/after the branch's prior HEAD:
```bash
go test ./model/feature/ -run 'TestOCCTBlendSimple' -count=1 -v 2>&1 \
  | grep -E '(PASS|FAIL): TestOCCTBlendSimple/' | sort > /tmp/after.txt
git stash && go test ./model/feature/ -run 'TestOCCTBlendSimple' -count=1 -v 2>&1 \
  | grep -E '(PASS|FAIL): TestOCCTBlendSimple/' | sort > /tmp/before.txt ; git stash pop
diff /tmp/before.txt /tmp/after.txt
```
Expected: exactly one line flips (`S1: FAIL→PASS`); PASS count 50→51; every other case identical.

- [ ] **Step 4: Full local suite + lint**

Run: `go test ./kernel/ops/ ./model/feature/ -count=1 && golangci-lint run kernel/ops/... model/feature/occtparity/...`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/fillet_runout_test.go
git commit -m "test(fillet): S1 runout imprint greens FAIL(area)→PASS, zero regression"
```

---

### Task 7: Generalize the solver to cone & elliptical-cylinder footprints (S4, T7)

S4's cone and T7's elliptical-cylinder footprints on the host plane are a circle and an ellipse respectively — same closed-form family as Task 3.

**Files:**
- Modify: `kernel/ops/fillet_runout_imprint.go`
- Test: `kernel/ops/fillet_runout_imprint_test.go`

**Interfaces:**
- Produces: `solveImprint` handles an ellipse footprint by mapping to the circle solve in the ellipse's axis frame; the cone footprint (a circle at the host plane) already works.

- [ ] **Step 1: Failing test** — ellipse footprint (semi-axes a=15,b=10) ∩ band line, assert the two crossings satisfy `(x/a)²+(y/b)²=1` within `1e-9`.
- [ ] **Step 2: Run** `go test ./kernel/ops/ -run TestSolveImprint -count=1 -v` → FAIL.
- [ ] **Step 3: Implement** the ellipse branch: affine-scale the band line and center into the unit-circle frame (`x/a, y/b`), reuse `lineCircleRoots`, map roots back; the outboard sub-arc uses the exact ellipse rational-quadratic NURBS + knot-insert already in `geom`.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Gate S4 + T7** — add `"S4","T7"` to `TestRunoutFamilyPasses`; run `go test ./model/feature/ -run 'TestOCCTBlendSimple/(S4|T7)' -v`; full zero-regression diff (PASS 51→53). Commit `feat(fillet): cone+ellipse runout footprints (S4,T7) green`.

---

### Task 8: Torus footprint (T1)

T1's torus base curve on the host plane is a circle — reuse the Task-3 circle solve; verify the torus footprint edge resolves to a circle in `geom` and no special-casing is needed.

**Files:**
- Modify: `kernel/ops/fillet_runout_imprint.go` (only if the torus footprint edge is not already a `geom.Circle`)
- Test: `kernel/ops/fillet_runout_imprint_test.go`, `model/feature/occtparity/fillet_runout_test.go`

- [ ] **Step 1: Failing test** — assert `solveImprint` on T1's torus footprint yields two crossings on the base circle.
- [ ] **Step 2: Run** → FAIL (or PASS if the circle path already covers it — then just add the gate).
- [ ] **Step 3: Implement** any torus-footprint-to-circle adaptation needed.
- [ ] **Step 4: Gate T1** — add `"T1"`; run `go test ./model/feature/ -run 'TestOCCTBlendSimple/T1' -v`; zero-regression diff (PASS 53→54). Commit `feat(fillet): torus-footprint runout (T1) green`.

---

### Task 9: B-spline footprint via bracketed root-find (T9)

T9's footprint is a general b-spline; no closed form. Solve footprint∩band by a 1-D bracketed root-find on the footprint's pcurve signed-distance to the band line.

**Files:**
- Modify: `kernel/ops/fillet_runout_imprint.go`
- Test: `kernel/ops/fillet_runout_imprint_test.go`, `model/feature/occtparity/fillet_runout_test.go`

**Interfaces:**
- Produces: `solveImprint` dispatches to `lineBSplineRoots(im, res)` when the footprint is a `geom.BSplineCurve`.

- [ ] **Step 1: Failing test** — a synthetic b-spline footprint crossing the band twice; assert two roots within `res.Weld()` of the true crossings; assert a grazing b-spline returns `ok=false`.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** `lineBSplineRoots`: evaluate the signed distance `b.signedDist(flat(footprint.PointAt(t)))` on a dense parameter sweep, bracket each sign change (larger than `res.Weld()`), refine each with bisection/Newton to `1e-4·scale`. Guard: fewer than 2 sign changes → `ok=false` (grazing/no-dip). The outboard sub-arc uses `geom` knot-insertion at the two roots.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Gate T9** — add `"T9"`; run `go test ./model/feature/ -run 'TestOCCTBlendSimple/T9' -v`; zero-regression diff (PASS 54→55). Commit `feat(fillet): b-spline-footprint runout (T9) marched-root green`.

---

### Task 10: Multi-crossing hardening + ADR-5 + full-suite close-out

S1 already exercises two bosses on two planes (handled as two `runoutImprint`s). Harden the >2-crossings and multi-footprint-per-face cases to honest-reject cleanly (fall back), record the decision, and run the whole suite.

**Files:**
- Modify: `kernel/ops/fillet_runout_detect.go` (honest-reject a face with >1 crossing footprint or a footprint with >2 band crossings — the fallback catches it)
- Modify: `docs/superpowers/specs/2026-07-13-curved-corner-miter-blend-engine-design.md` (ADR-5)
- Test: `kernel/ops/fillet_runout_detect_test.go`

- [ ] **Step 1: Failing test** — a face with a footprint crossing the band 4× (weaving) asserts `detectRunouts` omits it (returns no imprint for that host), so the fallback keeps the baseline.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** the guard (reuse `obstacleNodes`' `len(cs)!=2 ⇒ false` contract) and a `runoutImprint`-count guard per face.
- [ ] **Step 4: Write ADR-5** in the design spec: runout imprint = merge footprint into band + trim fillet, NO new faces, SEPARATE from ADR-4 obstacle rebuild; tiered footprint∩line solve (conic closed-form / b-spline marched); do-no-harm area-improved fallback; deferred: multi-weave footprints, non-coplanar runouts.
- [ ] **Step 5: Full local suite + lint + markdownlint + SPDX + zero-regression diff (PASS ≥55, five target cases flipped, all others identical). Commit** `feat(fillet): runout imprint multi-crossing guard + ADR-5; close thread-③ area cases`.

---

## Verification (whole slice)

- **Per family (unit):** each new function has a `kernel/ops` test with named fixtures; `solveImprint` verified against exact analytic crossings; `applyImprints`/`assembleFilletBody` verified against hand-computed analytic areas on synthetic slab+boss fixtures.
- **Per family (oracle):** `TestOCCTBlendSimple/<case>` within OCCT's 1% (`checkprops` deps). DRAWEXE dumps via `printf 'source <script>.tcl\n' | DRAWEXE -b` (never `-b <file>`; never line-by-line `foreach`).
- **Zero regression (mandatory each task):** full `TestOCCTBlendSimple` before/after diff — only the targeted case(s) flip; PASS count rises 50→55 across the slice; all others byte-identical.
- **Close-out:** `go test ./kernel/ops/ ./model/feature/ -count=1`; `golangci-lint run`; `markdownlint`; `scripts/add-spdx-headers.py --check`.

## Execution order

Task 1 (oracle localization) blocks Tasks 4–5. Tasks 2–3 (detect, circle solve) are parallel-safe with Task 1. Task 4→5→6 are sequential (apply → wire → gate S1). Tasks 7–9 (per-family solvers) are independent given the Task-6 scaffold. Task 10 closes out. Commit per task; NO PR until the whole corpus is green.
