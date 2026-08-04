<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Curved-Arm Trihedral Fillet Weld (Slice A · T5.1–T5.5) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Weld the already-built exact analytic arms (torus / cylinder) and the analytic-sphere corner of an axis-aligned Plane∧Cylinder trihedral fillet into a watertight, G1, topology-faithful solid — greening OCCT `tests/blend/simple` **B3 (20559.5), N1 (58091.9), O1 (65104.9)** and the `[Cylinder,Plane,Plane]` family to the DRAWEXE oracle (deps=1%).

**Architecture:** An all-analytic build (no NURBS fill, no marcher — proven by `.superpowers/sdd/m5-weld-setback-retrim-derivation.md`, oracle-closed on all 9 B3 faces). One fact drives it: the corner ball-centre **C is the common intersection of all three arm spines**, so each arm's setback station is the closed-form root of `spine(t)=C`; the weld rail where an arm meets the sphere is the arm's terminal radius-r cross-section, which is a **great circle of the corner sphere** (exact G0 + exact G1 since both normals are `(P−C)/r`). Nine result faces (3 trimmed arms + sphere spherical-triangle + 5 retrimmed hosts) each carry a single outer loop of exact `geom.Curve3` edges and are welded by the existing `assembleBody`. A **Gauss–Bonnet closure invariant** on the corner sphere fails loudly on a wrong traversal.

**Tech Stack:** Go, `kernel/ops` + `kernel/geom` (`geom.Torus/Cylinder/Sphere/Circle/Arc3d`), `kernel/topo`, `oblikovati.org/math`. Oracle: DRAWEXE at `../occt-build/lin64/gcc/bin/DRAWEXE`.

## Global Constraints

- **NO PR** — this milestone opens none; the whole OCCT corpus is not yet green. Accumulate + commit per task.
- Corpus-neutrality / count gate (the `-v` is REQUIRED):
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` — 54 until the weld greens, then **57** (B3/N1/O1). Untouched grids stay byte-identical (the weld is gated on a curved-arm `ef`; planar paths never change).
- DRAWEXE oracle is the source of truth. Self-contained per-case script (NOT the STEP-import path — its edges are `a_1_*`, not `s_N`): `../OCCT/tests/blend/simple/<CASE>`; run `printf 'source X.tcl\n' | DRAWEXE -b`; per-face areas via `explode result F; foreach f {sprops $f}`.
- SPDX `GPL-2.0-only` header on every new/changed `.go` (`python3 scripts/add-spdx-headers.py`).
- Funcs 4-20 lines; files <500 (`fillet.go` is 966 — do NOT grow it; new code in new focused files); explicit types (no `any`); early returns; ≤2 indent; no duplication; error messages carry the offending value + expected shape.
- Tolerances model-relative (ADR-0042): `res := ResolutionForBody(body)`, `res.Weld()`. **Corner-local** tolerances scale to the corner sphere radius r, NOT body diameter: endpoint/rail `res.Weld()*r`, spherical-triangle area `res.Weld()*r*r`. NO bare `1e-6`.
- **Never gate correctness on `IsSolid` alone** — a wrong-sign torus welds inside-out and passes `IsSolid`; every green gate pairs it with `Validate(...).HolesContained` AND a per-type faithfulness area AND (T5.5) a tessellated-volume regression.
- The do-no-harm floor stays the fallback: any station/closure/weld decline honest-rejects the whole op to the current clean error (`curvedArmUnweldedError`), never a partial body.

## The certified geometry (from the derivation — the oracle for every task)

B3 fixture frame (R=50, r=10, wedge x≥0,y≤0): three hosts **Wall** W (`x²+y²=2500`, `geom.Cylinder` axis ẑ), **Cap** K (`z=100`), **Radial** N (`x=0`); trihedral vertex V=(0,−50,100). Three arms + corner:

| Arm | Surface | Spine | Setback station (spine=C) | Assembled face area |
|-----|---------|-------|---------------------------|---------------------|
| P0 torus | `geom.Torus` major ρ=R−r=40, minor 10, axis ẑ, centre (0,0,90) | circle x²+y²=40², z=90 | major angle **φ\*=−75.522°** (cosφ\*=Cx/ρ=0.25) | **960.008** |
| P1 cyl | `geom.Cylinder` r=10, axis ẑ | line x=10, y=−38.7298 | axial **z\*=90** (also bottom foot-bite at z=0) | **1641.13** |
| P2 planar cyl | `geom.Cylinder` r=10, axis ŷ | line x=10, z=90 | axial **y\*=−38.7298** | **608.367** |
| Corner | `geom.Sphere` r=10, centre **C=(10,−38.7298,90)** | point C | — | **182.348** |

Corner ball-centre **C=(10,−38.7298,90)** (verified 10²+38.7298²=1600=40²). Host-tangent points (sphere↔host, = spherical-triangle vertices): **T_W=(12.5,−48.412,90)**, **T_K=(10,−38.7298,100)**, **T_N=(0,−38.7298,90)**. Rail unit dirs from C: n_W=(0.25,−0.96825,0), n_K=(0,0,1), n_N=(−1,0,0). Spherical triangle interior angles **90°/90°/104.478°**, excess E=arccos(−0.25)=1.82348, area r²E=**182.348**.

**Nine result faces (each one outer loop of exact curves; welded by `assembleBody`):**

| Face | Boundary edges (type) | Welds (shared rail) | Area |
|------|----------------------|---------------------|------|
| Wall W | bottom arc R50 (z=0), P1 ruling (line), **torus arc R50 z90**, y=0 edge | P1, P0 | 5931.52 |
| Radial y=0 | rect − torus corner-arc − planar corner-arc | P0, P2 | 4957.08 |
| Radial x=0 (N) | rect y∈[−38.73,0]×z∈[0,90]: P1 line + P2 line | P1, P2 | 3485.69 |
| Bottom cap z=0 | quarter disk − P1 foot-bite | P1 | 1932.47 |
| **Cyl arm P1** | 2 host rulings (W,N) + bottom foot arc + **top setback great-arc** | W,N; sphere | 1641.13 |
| **Torus arm P0** | wall arc R50 + cap arc R40 + y=0 far end + **setback great-arc** | W,K; sphere | 960.008 |
| Top cap K | **torus arc R40** + P2 line (x=10) + apex runout + y=0 edge | P0, P2 | 860.844 |
| **Planar arm P2** | 2 host lines (K,N) + apex end + **setback great-arc** | K,N; sphere | 608.367 |
| **Sphere** | 3 great-arcs (rails to P0,P1,P2) | P0,P1,P2 | 182.348 |

Σ=20559.4 (oracle 20559.5). **Non-obvious:** the through-going cyl arm P1 retrims BOTH ends (top setback into sphere + bottom foot-bite → bottom cap is 1932.47, not the naive quarter-disk 1963.50).

---

## File Structure

- **Create `kernel/ops/fillet_curved_corner_solve.go`** (T5.1) — the corner solver: gather the ≤3 curved arms + sphere at a shared trihedral vertex; `spine=C` station per arm; the host-tangent points; the Gauss–Bonnet closure certificate. Pure geometry, unit-testable without assembly.
- **Create `kernel/ops/fillet_curved_rails.go`** (T5.2) — the weld great-arc rails (arm terminal cross-section = sphere great circle) + G1 assertion helper.
- **Create `kernel/ops/fillet_curved_retrim.go`** (T5.3) — the retrimmed host outer loops with **circular-arc** contact rails (wall R50/z90, cap R40/z100) that `transformFace`'s straight-tangent pull cannot emit.
- **Create `kernel/ops/fillet_curved_weld.go`** (T5.4) — assemble the nine `filletFace`s (3 arms + sphere + retrimmed hosts) and route the convex axis-aligned family through it from `filletResolvedEdges`, keeping `curvedArmUnweldedError` as the do-no-harm fallback.
- **Modify `kernel/ops/fillet.go`** (T5.4) — in `filletResolvedEdges`, replace the unconditional floor reject with: try the curved weld; on decline, fall back to the floor error.
- **Tests:** `kernel/ops/fillet_curved_corner_solve_test.go`, `fillet_curved_weld_test.go`, and the corpus gate in `model/feature` (`TestOCCTBlendSimple`).

---

## Task T5.1: Corner solver — setback stations + Gauss–Bonnet closure

**Files:** Create `kernel/ops/fillet_curved_corner_solve.go`, `kernel/ops/fillet_curved_corner_solve_test.go`.

**Interfaces:**
- Consumes: the solved `[]edgeFillet` (curved arms carry `ef.armSurface geom.Surface` = `geom.Torus` or `geom.Cylinder`; `ef.a/ef.b` are the two host faces; `ef.edge`), the corner `geom.Sphere` (built by Task 4's `cylinderHostCorner`; its `Center` = C, `Radius` = r), and `res Resolution`.
- Produces:
  ```go
  type armSetback struct {
      arm      geom.Surface // the ef.armSurface (Torus or Cylinder)
      station  float64      // spine parameter where spine(t)=C (torus: major angle; cyl: axial)
      railDir0 math.UnitVector3 // unit (T_hostA − C)/r
      railDir1 math.UnitVector3 // unit (T_hostB − C)/r
  }
  type cornerWeld struct {
      center  math.Point3 // C
      radius  float64     // r
      arms    []armSetback
      tPoints []math.Point3 // host-tangent points (spherical-triangle vertices)
  }
  // solveCurvedCorner gathers the curved arms + sphere at a shared trihedral vertex and solves the
  // setback stations + host-tangent points; returns false (honest-reject) if fewer than 3 arms meet,
  // a station has no in-domain root (gap), or the closure certificate fails.
  func solveCurvedCorner(sphere geom.Sphere, arms []edgeFillet, res Resolution) (cornerWeld, bool)
  // curvedClosureValid enforces the four fail-loud invariants (§A.3).
  func curvedClosureValid(w cornerWeld, res Resolution) bool
  ```

- [ ] **Step 1: Failing test — torus station.** `TestSolveCurvedCorner_B3Stations`: build the B3 torus arm (`geom.NewTorusWithRef((0,0,90), ẑ, x̂, 40, 10)`), cyl arm (r10 axis ẑ about line x=10,y=−38.7298), planar cyl arm (r10 axis ŷ about line x=10,z=90), and `geom.NewSphere((10,−38.7298,90),10)`. Assert `solveCurvedCorner` returns ok and the three stations: torus major-angle **−75.522°±1e-3 rad**, cyl axial **z=90±res.Weld·r**, planar axial **y=−38.7298±res.Weld·r**. Run → FAIL (function undefined).

- [ ] **Step 2: Implement the station solver.** For each arm, solve `spine(t)=C` in closed form:
  - torus: `cosφ*=(C−centre)·refDir/ρ`, `sinφ*=(C−centre)·(axis×refDir)/ρ`, `φ*=atan2(sin,cos)`; reject if `|‖C−centre‖_inPlane − ρ| > res.Weld()*R` (C not on spine).
  - cylinder: project C onto the axis line → axial station; reject if `dist(C, axisLine) > res.Weld()*R`.
  Host-tangent point for a host h = `C + r * unit(projectionOfC-onto-h-contact-direction)`; equivalently the sphere's tangent point with host h (for a plane: `C + r*(−n̂_h)` toward the plane; for the wall cylinder: `C + r*unit(C_xy)` radially outward to radius R). Assert each `T` lies on its host (`|dist − hostRadius| ≤ res.Weld()*R` for the wall; on-plane for planes). Run Step 1 → PASS.

- [ ] **Step 3: Failing test — closure.** `TestCurvedClosure_B3`: from the solved `cornerWeld`, assert the spherical triangle over {n_W,n_K,n_N} has interior angles {π/2, π/2, arccos(−0.25)} (±1e-4) and area `r*r*E = 182.348 ± res.Weld()*r*r`; and assert `curvedClosureValid` returns true. Then a **negative** case: perturb one railDir by 5° and assert `curvedClosureValid` returns false (the guard bites). Run → FAIL.

- [ ] **Step 4: Implement `curvedClosureValid`** (the four §A.3 invariants): (1) the three rails chain `T_W→T_K→T_N→T_W` with shared endpoints (‖gap‖≤res.Weld·r); (2) each rail subtense == geodesic `arccos(n_i·n_j)` between its host-tangent points; (3) signed spherical excess E∈(0,2π) and `r²E` == forward triangle area; (4) each arm has exactly one in-domain station root. Any failure → false. Run Step 3 → PASS.

- [ ] **Step 5:** `go build ./... && go vet ./kernel/... && gofmt -l kernel/ops` clean; `go test ./kernel/ops -run 'CurvedCorner|CurvedClosure'`; corpus prints **54** (nothing wired yet). Commit `feat(blend): curved-arm corner setback-station solver + Gauss-Bonnet closure guard`.

---

## Task T5.2: Weld great-arc rails (arm ↔ sphere, exact G1)

**Files:** Create `kernel/ops/fillet_curved_rails.go`, extend `fillet_curved_weld_test.go` (new file).

**Interfaces:**
- Consumes: `cornerWeld` (T5.1), the corner `geom.Sphere`.
- Produces:
  ```go
  // curvedSetbackRail is the great-circle arc on the corner sphere between an arm's two host-tangent
  // points — the shared curve welding that arm's setback end to the sphere (exact G0), along which
  // both surface normals are (P−C)/r (exact G1).
  func curvedSetbackRail(w cornerWeld, arm armSetback) (geom.Arc3d, bool)
  // curvedRailG1 samples the rail and asserts ‖n_arm(P) − n_sphere(P)‖ ≤ res.Weld() (both (P−C)/r).
  func curvedRailG1(arm geom.Surface, rail geom.Arc3d, center math.Point3, r float64, res Resolution) bool
  ```

- [ ] **Step 1: Failing test.** `TestCurvedSetbackRail_B3`: for the torus arm, assert `curvedSetbackRail` returns a `geom.Arc3d` from **T_W to T_K** whose supporting circle is centred at **C** with radius **r=10** and lies in a plane through C (a great circle), subtense **90°**; for the cyl arm, T_W→T_N, subtense **104.478°**; for the planar arm, T_K→T_N, subtense **90°**. Run → FAIL.

- [ ] **Step 2: Implement `curvedSetbackRail`** — the arc through {T_hostA, midpoint, T_hostB} where the midpoint is `C + r*unit(n_A + n_B)` (the great-circle bisector), via `geom.Arc3dByThreePoints`. Assert (return false otherwise) the circle centre ≈ C and radius ≈ r (it is a great circle by construction; the check catches a mis-built tangent point). Run Step 1 → PASS.

- [ ] **Step 3: G1 test + impl.** `TestCurvedRailG1_B3`: assert `curvedRailG1` true for all three arms; and a mutation — offset the arm surface centre by `0.1*r` and assert it returns false. Implement by sampling ~5 points along the rail, computing both normals as `(P−C)/r` (arm normal via the canal identity; sphere normal via `sphere.NormalAt`), asserting agreement ≤ res.Weld(). Run → PASS.

- [ ] **Step 4:** build/vet/gofmt clean; `go test ./kernel/ops -run 'CurvedSetbackRail|CurvedRailG1'`; corpus **54**. Commit `feat(blend): curved-arm setback great-arc weld rails with exact-G1 assertion`.

---

## Task T5.3: Curved-rail host retrim

**Files:** Create `kernel/ops/fillet_curved_retrim.go`, extend `fillet_curved_weld_test.go`.

**Interfaces:**
- Consumes: `cornerWeld` (T5.1) + `res`. Each host face is `ef.a`/`ef.b` from the arms.
- Produces per-host retrimmed outer loops as `filletFace` (surface = the host's original `geom.Surface`; loop = exact-curve boundary). The circular-arc contact rails are the crux (`transformFace` only pulls a straight tangent vertex):
  ```go
  // curvedHostArc returns the circular contact rail where a torus arm meets a host: on the WALL it is
  // the circle radius R in plane z = C_z (azimuth span = the arm's φ range); on the CAP it is the circle
  // radius R−r in the cap plane. side selects which.
  func curvedHostArc(host geom.Surface, tor geom.Torus, w cornerWeld, res Resolution) (geom.Arc3d, bool)
  // retrimCurvedHost builds one retrimmed host face's outer loop from its original edges minus the
  // corner bite plus the arm contact rails (circular arcs for torus-adjacent hosts, straight rulings
  // /lines for cyl/planar-adjacent hosts), per the 9-face table.
  func retrimCurvedHost(host *topo.Face, w cornerWeld, res Resolution) (filletFace, bool)
  ```

- [ ] **Step 1: Failing test — wall arc.** `TestCurvedHostArc_B3Wall`: assert `curvedHostArc` for the wall returns a `geom.Arc3d` on the circle **centre (0,0,90), radius 50, plane z=90**, azimuth **0°→−75.522°**; for the cap, circle **centre (0,0,100), radius 40, plane z=100**. Run → FAIL.

- [ ] **Step 2: Implement `curvedHostArc`.** Wall: circle centre = (axis point at height C_z), radius = host cylinder R, in plane ⊥ axis; endpoints at azimuth 0 (far, the y=0 cut) and the arm φ\* — read from `w`. Cap: circle centre = axis∩cap, radius = R−r, endpoints likewise. Run Step 1 → PASS.

- [ ] **Step 3: Retrim-loop test + impl.** `TestRetrimCurvedHost_B3`: assert the retrimmed **Radial x=0** loop is the rectangle y∈[−38.7298,0]×z∈[0,90] (area 3485.69±0.1 via a loop-area helper), the **Wall** loop area ≈5931.52, the **Cap** loop ≈860.844 (build a small polygonal/analytic area check per loop). Implement `retrimCurvedHost` by assembling each host's boundary per the 9-face table: original outer edges, cut back at the arm/sphere contact, plus the arm contact rails (circular arc from Step 2 for torus-adjacent, straight ruling/line otherwise). Run → PASS.

- [ ] **Step 4: Both-ends note.** Ensure the through-going cyl arm P1's foot-bite on the **bottom cap z=0** is applied (bottom cap → 1932.47, not 1963.50). Add an assert for the bottom-cap area. build/vet/gofmt clean; corpus **54**. Commit `feat(blend): circular-arc curved-host retrim for the trihedral fillet corner`.

---

## Task T5.4: Assemble the nine faces + route the family through the weld

**Files:** Create `kernel/ops/fillet_curved_weld.go`; modify `kernel/ops/fillet.go` (`filletResolvedEdges`) and `kernel/ops/fillet_curved_assemble.go`.

**Interfaces:**
- Consumes: T5.1 `solveCurvedCorner`, T5.2 rails, T5.3 retrim; the solved `[]edgeFillet`, `body`, `blends`.
- Produces:
  ```go
  // assembleCurvedArmBody builds the nine result faces (3 trimmed arms + sphere spherical-triangle +
  // retrimmed hosts) and welds them via assembleBody. Returns false (→ do-no-harm floor) on any
  // solve/closure/weld decline — never a partial body.
  func assembleCurvedArmBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) (*topo.Body, bool)
  // curvedArmTrimmedFace emits one setback arm face: the analytic arm surface trimmed to its assembled
  // span, outer loop = host contact rails + setback great-arc.
  func curvedArmTrimmedFace(arm armSetback, w cornerWeld, hosts [2]*topo.Face) filletFace
  ```
- The corner sphere face reuses the spherical-triangle loop (three setback great-arcs from T5.2); surface = the `geom.Sphere`.

- [ ] **Step 1: Failing test — B3 solid + faithful.** `TestFilletEdges_B3CurvedArmWeld` in `fillet_curved_weld_test.go`: `body := filletedCorpusEdges(t,"simple/B3",10)`; assert `body.IsSolid()`, `Validate(body).Valid && .HolesContained`, and per-type faithfulness (count via `countSurfaceFacesNear[T]`): exactly ONE `geom.Torus` face area≈**960.008**(±5), ONE `geom.Sphere` area≈**182.348**(±2), and the two `geom.Cylinder` arm faces ≈**1641.13** and **608.367**. Run → FAIL (floor still rejects).

- [ ] **Step 2: Implement `assembleCurvedArmBody`.** Gather the curved arms + their corner sphere (from `blends`/Task-4 corner), call `solveCurvedCorner`; build the 3 arm `filletFace`s (`curvedArmTrimmedFace`), the sphere face, and the retrimmed host faces (`retrimCurvedHost` for arm-adjacent hosts, `transformFace` unchanged for the rest); `assembleBody(faces,"fillet")`. Return false on any `ok==false`.

- [ ] **Step 3: Route it.** In `filletResolvedEdges` (`fillet.go`), where the floor currently does `if curvedArmFils(fils) { return nil, curvedArmUnweldedError(fils) }`: first try `if b, ok := assembleCurvedArmBody(body, fils, blends); ok { validate → return b }`; only on `!ok` return `curvedArmUnweldedError` (do-no-harm). The planar path is untouched (guard: only reached when `curvedArmFils(fils)`). Run Step 1 → PASS.

- [ ] **Step 4: Do-no-harm + corpus.** Confirm a case that classifies a curved arm but cannot weld still returns the clean floor error (no panic, no partial body). Corpus: `TestOCCTBlendSimple` → **B3 flips to PASS**; run the full count → **55**; all other grids unchanged (base-vs-head diff = only B3 FAIL→PASS). build/vet/gofmt clean; full `go test ./kernel/ops`. Commit `feat(blend): weld the axis-aligned curved-arm trihedral fillet into a watertight solid (greens B3)`.

---

## Task T5.5: Oracle gate — N1/O1, family, volume regression

**Files:** extend `kernel/ops/fillet_curved_weld_test.go`; `model/feature` corpus is data-driven (no code change expected).

- [ ] **Step 1: N1/O1 (radius 5).** Confirm `TestOCCTBlendSimple/N1` and `/O1` flip to PASS within 1% of **58091.9** / **65104.9**. If they need structural faithfulness asserts (r=5: torus major 45/minor 5, cyl r5), add them in the fixture frame; else the area gate suffices. Reconcile any per-face difference with the derivation's r=5 numbers (torus φ range, cyl subtense 96.379°). Corpus → **57**.

- [ ] **Step 2: Volume/manifold regression (the wrong-sign guard).** `TestFilletEdges_B3VolumeRegression`: assert the tessellated `body` volume matches the OCCT/analytic value (a wrong-sign torus welds inside-out — passes `IsSolid`, fails volume). Use the existing tessellated-volume helper; gate to 1%.

- [ ] **Step 3: Family.** Extend to B7 (43467.9, r10), L8 (61663.5, r5), M5 (61187.1, r5), N7 (61222.9, r5), H7 (554732, r10) as each is a `[Cylinder,Plane,Plane]` axis-aligned trihedral. For any that is oblique/ellipse (config iii) or otherwise unsupported, confirm it rides the do-no-harm floor (clean reject), not a regression. Record the final corpus count.

- [ ] **Step 4: M4 tripwire.** Confirm the M4 torus-band tessellation cases (T1/T4) and S1/S4/T7/S7 are byte-identical (the weld touches only curved-ARM efs, not the setback/runout path). build/vet/gofmt/full-suite clean. Commit `test(blend): oracle-gate curved-arm trihedral family + volume regression`.

---

## Self-review notes
- Every setback station, rail subtense, closure area, and face area in the tasks is a DRAWEXE-oracle value from the derivation — no invented numbers.
- The weld is gated on `curvedArmFils(fils)` (a curved-arm `ef` present); the planar/setback/runout paths are never entered, so corpus non-regression is structural, not luck.
- Do-no-harm: `assembleCurvedArmBody` returns `(body,false)` on any decline and `filletResolvedEdges` falls back to `curvedArmUnweldedError` — the floor from the prior task remains the safety net; no path ships a partial solid.
- Carried Minors (fold into the pre-PR cleanup, not this milestone): Task-4 concave-sign hard-code, T5.0 concave `flat` orientation, M3/M4 batched cleanup.
