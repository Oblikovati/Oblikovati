<!-- SPDX-License-Identifier: GPL-2.0-only -->

# General far-runout engine + sphere-slice completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use
> checkbox (`- [ ]`) syntax. Implementers run on OPUS; reviewers/planning on FABLE (user directive:
> bulletproof, no compromise, same rigor as N7).

**Goal:** Build the general far-runout engine (`armFarRunout`) — a fillet arm's far terminus = `arm ∩
capping-face` — as the reusable capability that unblocks the sphere-host corners now and the cone/torus
families later, then use it to green OCCT `tests/blend/simple/{D5, D9, E4}` (corpus 57→**60**). B3 and the
57 existing greens stay BYTE-IDENTICAL (perpendicular caps route to the existing `farCrossSectionArc`
verbatim); N7's canal far-runout is left untouched (a documented sibling, strangled onto the engine only
later, behind its own byte-diff gate).

**Architecture:** `.superpowers/sdd/far-runout-engine-architecture.md` (ADR-1..4). Port math (DRAWEXE-
grounded): `.superpowers/sdd/far-runout-port-math.md`. Sphere corner/arm: `.superpowers/sdd/sphere-host-
corner-derivation.md`. `armFarRunout` is the routing layer wherever `armRailBundle` calls
`farCrossSectionArc` today; it OWNS the whole far termination `(h0′, h1′, runout)` (under an oblique cap
the host rails' outer ends move — the N7 W4 lesson). The intersection MATH lives behind a port
`intersectArmCapping`; trims are analytic-on-the-arm, feet always closed-form.

**Tech Stack:** Go (`oblikovati` GPL, `kernel/ops` + `kernel/geom`). Reuse `farCrossSectionArc`,
`farArcsBiting`, `farRunoutFace`, `spliceCornerBite`, `armRailBundle`, `geom.SpiricArc`/`TorusSectionCoeffs`,
`planeSphereCurve`, `TraceSurfaceIntersection`, `planePerpToDir`. DRAWEXE oracle.

**Base:** `48642784` (the corpus-neutral SP3 WIP: sphere pinch tangent-point + per-arm torus station),
corpus 57. Slices SP1/SP2 (torus arm + sphere corner) are landed & reviewed.

## Global Constraints

- **NO PR until the whole corpus is green** (targets 57→60; accumulate + commit per task).
- **B3 + all 57 existing corpus greens BYTE-IDENTICAL.** ADR-2: the perpendicular branch of `armFarRunout`
  calls `farCrossSectionArc` VERBATIM with the exact current arguments → byte-identity by CALL-GRAPH
  (not numerics). Certify the F3a way: (a) a population probe — every current far vertex is perpendicular
  (`|n̂_cap·t̂| > 1−1e-12`) and the new oblique cases are `< 1−1e-3` (≥9 orders apart); (b) a full-corpus
  worktree verdict-set byte-diff (195 cases) unchanged except the target cases. N7's canal far-runout is
  NOT touched.
- **Zero tuned constants.** Every trim/foot/ellipse/spiric is a closed form of the arm + capping geometry;
  area gates CHECK the OCCT oracle, never fit. `res.Weld·scale` tolerances (ADR-0042); the perpendicular
  dispatch reuses the existing `sinFloor`/`planePerpToDir` dimensionless gate; no bare 1e-6.
- **Honest reject, never a forced green.** The engine handles the TRIHEDRAL far vertex (1 filleted edge +
  1 capping face + 2 sharp edges). n-valent / ≥2 non-host faces / fillet-fillet interference / no
  intersection between the feet / grazing → the do-no-harm floor with the exact obstruction. NEVER loosen
  a gate; NEVER ship a mis-closed shell.
- **Shared-edge identity:** the arm face's far edge and the capping-face bite sample the SAME `runout.trim`
  curve identically (the watertightness mechanism). Trims are analytic-on-the-arm (never polylines).
- **Functions 4–20 lines (funlen 30/20); files < 500; SPDX; explicit types; errors carry value + shape;
  `math.NewPoint3` → `math.P3`.**
- **Corpus count (`-v` REQUIRED):** `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE
  '^\s*--- PASS: TestOCCTBlendSimple/'`.
- **DRAWEXE oracle:** `source test-utilities/occt-blend/oracle/drawenv.sh`; D5 recipe `psphere s 15 -60 60
  90; trotate s 0 0 0 0 0 1 90; tscale s 0 0 0 10; explode s E; blend result s 10 s_2 10 s_1 10 s_6`.

## The oracle-verified port math (the spec for FR1/FR2 — from the derivation)

- **Dispatch (per-arm-per-far-vertex, inside `armFarRunout`):** perpendicular iff the capping face is a
  Plane AND `|n̂_cap · t̂_spine(F)| > 1 − sinFloor` (reuse `planePerpToDir`). Perpendicular → the EXISTING
  `farCrossSectionArc` (byte-identical). Oblique → the port.
- **Capping face** = the unique non-host transverse face at F (`cappingFaceAtFarVertex`): decline if ≥2
  non-host faces or a second picked edge at F (the n-valent setback regime, out of scope).
- **`intersectArmCapping(arm, capping geom.Surface, feet [2]math.Point3, r, res) (geom.Curve3, bool)`:**
  - **Torus ∩ Plane** (D5 workhorse): `geom.SpiricArc`/`TorusSectionCoeffs` — covers BOTH axis-parallel
    (`C=m̂·â=0`) and general oblique; degeneracy `M→0` (plane ⊥ torus axis) → exact latitude circle.
    DRAWEXE residual 6.83e-7 (= OCCT's own approx error). This + Cyl∩Plane are all the SPHERE slice needs.
  - **Cyl ∩ Plane** (oblique): exact ellipse `z(θ) = (m̂·(o−O) − R_a(m_r cosθ + m_b sinθ)) / (m̂·â)`.
  - **Torus ∩ Sphere** (closed form `u(v)=Ψ±arccos h(v)`, DRAWEXE 1.25e-7) + **∩Cone/∩Cyl** (per-v
    quadratic → feet-bracketed Newton on the arm chart, DRAWEXE 5.80e-8): DERIVED + validated; ship in the
    port for the future cone/torus families but NOT exercised by the sphere slice (the census: D5/D9/E4
    have no ∩sphere/∩cone/∩cyl capping).
  - **Feet (closed-form, engine-supplied):** each arm's spring/contact curves are analytic (torus arm =
    u-circles at `v_P=+π/2` (plane host) / `v_S=atan2(−h,m_R)` (sphere host); cyl arm = rulings). Feet =
    spring ∩ capping = `atan2 ± arccos` (circle∩plane/sphere) or linear/quadratic (ruling). D5 feet hit
    the oracle to 3.9e-5 / 8.2e-8.
  - **Branch selection:** enumerate the ± signs; the correct branch has the SAME sign matching BOTH feet
    to `res.Weld·scale` (D5: 8e-8 vs 1.5e2, 11 orders apart); a midpoint material-side certificate rejects
    the mirror oval; 0 or ≥2 survivors → decline.
  - **Pitfalls:** existence-between-feet (closed-form section extrema, decline if the section doesn't reach
    between the feet or grazes below weld·scale); foot-at-a-section-extreme (spring ∥ capping → degenerate
    foot, decline — never pick a sign).

## File Structure
- `kernel/ops/fillet_far_runout.go` (create) — `armFarRunout` (dispatch + `runout` object +
  `cappingFaceAtFarVertex` scope guard) + the perpendicular verbatim call.
- `kernel/ops/fillet_intersect_arm_capping.go` (create) — the `intersectArmCapping` port (Torus∩Plane,
  Cyl∩Plane; Torus∩Sphere; ∩Cone/∩Cyl marched) + the spring curves + closed-form feet + branch selection.
- `kernel/geom` — only if a spiric/ellipse section helper is missing (reuse `SpiricArc`/`TorusSectionCoeffs`
  first).
- `kernel/ops/fillet_curved_weld.go` / `armRailBundle` (modify) — route the far terminus through
  `armFarRunout` (the engine owns `h0′,h1′,runout`).
- `kernel/ops/fillet_curved_farrunout.go` (modify) — generalize `spliceCornerBite`/`biteArcBulge`/
  `segPolyline` to an arbitrary analytic `trim` curve (Arc3d path kept verbatim; the oblique bite is a new
  branch by capping-face identity); extract `reverseChainSeg`.
- SP3-completion: the sphere-host retrim (S7 gap) + assembly; the corpus test D5/D9/E4 gates.
- Tests alongside each.

Task order — **FR1 → FR2 → FR3 → FR4 → FR5.** FR1 lands the engine skeleton byte-identical (everything
routes to the fast-path); FR2 fills the port math; FR3 wires the engine + retrim generalization; FR4 uses
it + the sphere-host retrim to green D5/D9/E4; FR5 gates + tessellation.

---

### Task FR1: `armFarRunout` engine skeleton + dispatch + scope guard (byte-identical)

**Files:** Create `kernel/ops/fillet_far_runout.go` (+ `_test.go`); read `armRailBundle`/`farCrossSectionArc`.

**Interfaces:**
- Produces: `type armRunout struct { trim endSeg; feet [2]math.Point3; capping *topo.Face; regime runoutRegime }`;
  `armFarRunout(arm armFillet, farVertex *topo.Vertex, res Resolution) (h0, h1 endSeg, run armRunout, ok bool)`
  — SKELETON: `cappingFaceAtFarVertex` scope guard; the perpendicular dispatch (`|n̂_cap·t̂| > 1−sinFloor` +
  Plane) → call `farCrossSectionArc` VERBATIM (exact current args), returning the current outer ends as
  `h0/h1` + `run.trim` = the cross-section arc; the oblique branch returns `ok=false` (a floor placeholder
  — FR2/FR3 fill it). `cappingFaceAtFarVertex(farVertex, arm) (*topo.Face, bool)` — the unique non-host
  transverse face; decline on ≥2 non-host / second-picked-edge.
- `intersectArmCapping(...)` interface declared (stub returns false).

- [ ] **Step 1 — Failing test:** the perpendicular dispatch on a current perpendicular far vertex (B3 or a
  simple case) returns the SAME cross-section arc `farCrossSectionArc` produces today (call-graph identity —
  assert the returned curve/points are what the existing path yields); `cappingFaceAtFarVertex` finds the
  unique cap and declines the ≥2-non-host case. Population probe: D5's meridian far vertex is OBLIQUE
  (`|n̂_cap·t̂| < 1−1e-3`), the current perpendicular cases are `> 1−1e-12`.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the skeleton + dispatch + scope guard.
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus `57` byte-identical (armFarRunout is not yet wired into the weld — this is the
  engine landing; OR if wired as a pure passthrough, all current cases take the verbatim fast-path → 57
  byte-identical, full verdict-set worktree diff); B3/N7 unchanged; build/vet/gofmt/lint clean.**
- [ ] **Step 6 — Commit** (`feat(ops): armFarRunout engine skeleton + perpendicular fast-path dispatch (FR1)`).

---

### Task FR2: `intersectArmCapping` port — Torus∩Plane + Cyl∩Plane + feet + branch selection

**Files:** Create `kernel/ops/fillet_intersect_arm_capping.go` (+ `_test.go`).

**Interfaces:**
- Consumes: `geom.SpiricArc`/`TorusSectionCoeffs`, `planeSphereCurve`, the torus/cylinder/plane geometry,
  `res.Weld`.
- Produces: `intersectArmCapping(arm, capping geom.Surface, feet [2]math.Point3, r, res) (geom.Curve3, bool)`
  — Torus∩Plane (spiric, both attitudes) + Cyl∩Plane (ellipse), analytic-on-the-arm, restricted to the
  arc between `feet`, branch-selected; plus `armSprings(arm) [2]geom.Curve3` and `springCapFeet(spring,
  capping) (math.Point3, bool)` (the closed-form feet). Torus∩Sphere (closed-form u(v)) + ∩Cone/∩Cyl
  (feet-bracketed Newton) implemented but flagged un-exercised (future families) — include if clean, else
  a follow-on; the SPHERE slice needs only Torus∩Plane + Cyl∩Plane.

- [ ] **Step 1 — Failing test (oracle-pinned):** Torus∩Plane on D5's meridian arm + its bottom-cap plane →
  the trim curve matches DRAWEXE's meridian far edge to `res.Weld·scale` (~6.83e-7, the OCCT approx error);
  the feet match (D5 3.9e-5 / 8.2e-8); the branch selection picks the material-side oval (mutation: force
  the mirror oval → the midpoint material certificate rejects it). Cyl∩Plane oblique → the exact ellipse
  (vs OCCT's `intersect` centre/radii). Honest-reject a grazing/no-intersection config.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the port (helpers: spiric-coeffs, ellipse, springs, feet, branch-select).
- [ ] **Step 4 — Run tests; verify pass** (oracle residuals reported).
- [ ] **Step 5 — build/vet/gofmt/lint clean; corpus `57` (port not yet wired).**
- [ ] **Step 6 — Commit** (`feat(ops): intersectArmCapping port — torus∩plane spiric + cyl∩plane ellipse (FR2)`).

---

### Task FR3: wire the engine into the arm-weld + generalize the bite (shared-edge identity)

**Files:** Modify `fillet_curved_weld.go`/`armRailBundle` (route the far terminus through `armFarRunout`);
`fillet_curved_farrunout.go` (`spliceCornerBite`/`biteArcBulge`/`segPolyline` generalization + extract
`reverseChainSeg`). Tests.

**Interfaces:** `armRailBundle` calls `armFarRunout` (gets `h0′,h1′,run`); the arm face's far edge =
`run.trim`; the capping-face bite via the generalized `spliceCornerBite` on `run.trim` (same curve object,
shared-edge identity). The engine's `h0′,h1′` replace the old outer ends (equal to them in the
perpendicular case → byte-identical).

- [ ] **Step 1 — Failing test:** for a perpendicular case, `armRailBundle` via `armFarRunout` produces
  BYTE-IDENTICAL faces to today (the verdict-set diff). For D5's oblique meridian arm, the arm face's far
  edge = the spiric `run.trim`, and the capping-face (bottom-cap) bite is the SAME spiric curve
  (point-identical, shared-edge). `spliceCornerBite` handles the analytic spiric trim (Arc3d path
  unchanged for existing cases).
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the wiring + the bite generalization (Arc3d path verbatim; oblique branch by
  capping identity).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus `57` byte-identical (full verdict-set worktree diff — the perpendicular fast-path +
  the verbatim Arc3d bite keep B3/N7/all-57 unchanged; D5/D9/E4 now floor deeper — on the sphere-host
  retrim, FR4); B3/N7 goldens unchanged; lint clean.**
- [ ] **Step 6 — Commit** (`feat(ops): route arm far-runout through the engine + generalize the bite (FR3)`).

---

### Task FR4: sphere-host retrim (S7 gap) + whole-body assembly → green D5/D9/E4 (corpus 57→60)

**Files:** Modify the host retrim for the sphere host (the S7 gap flagged in the sphere derivation) +
assembly; the corpus test D5/D9/E4 gates. Honest-report any weld obstruction.

**Interfaces:** Consumes SP1/SP2 (torus arms + sphere corner) + FR1–FR3 (the far-runout engine gives the
oblique arm far termini) + the existing weld/retrim; the sphere-corner trim = the spherical-triangle rail
loop with the degenerate pinch vertex.

- [ ] **Step 1 — Failing (whole-body) test:** `TestOCCTBlendSimple/{D5,D9,E4}`: Σ = the corpus oracle
  (D5 134780; D9/E4 per `corpus.json`) within `deps`, all oracle faces, WATERTIGHT (Valid +
  HolesContained + IsSolid; every edge 2-incident), volume, per-face areas (corner sphere 55.7891 for D5,
  the torus arms). Currently floors on the sphere-host retrim.
- [ ] **Step 2 — Run, verify it fails.**
- [ ] **Step 3 — Implement** the sphere-host retrim (extend the retrim/`curvedHostFaces` for the geom.Sphere
  host — the S7-gap coverage; reuse the existing machinery + the FR3 bite) + the assembly + the degenerate
  pinch-vertex handling. Honest-reject with the exact obstruction if a seam won't close.
- [ ] **Step 4 — Run; verify D5/D9/E4 green.** Corpus prints `60`.
- [ ] **Step 5 — Non-regression:** all 57 prior greens byte-identical; B3/N7/M5 unchanged; full suite + lint.
- [ ] **Step 6 — Commit** (`feat(ops): green OCCT blend/simple/{D5,D9,E4} via the far-runout engine + sphere retrim (corpus 57→60)`).

---

### Task FR5: DRAWEXE gate + tessellation + non-regression

- [ ] **Step 1 —** Re-run the DRAWEXE D5/D9/E4 recipes; confirm Σ + per-face + the spiric far edges vs ours.
- [ ] **Step 2 —** Tessellation check (CLAUDE.md priority): the torus arms + sphere corner + the spiric
  far-runout faces mesh to their true areas, no folds (apply the N7 trimmed-sub-span lesson if any canal/
  arm sub-edge presents whole-curve geometry).
- [ ] **Step 3 —** Full `go test ./...` + lint + markdownlint; whole corpus byte-identical except D5/D9/E4;
  coverage > 80% on the new engine/port files.
- [ ] **Step 4 — Commit** (fold into FR4 if no new files).

---

## Verification
- **D5/D9/E4:** Σ = oracle, all faces, watertight, volume, per-face (spiric far edges + torus arms +
  sphere corner) — corpus 57→60; DRAWEXE-confirmed; tessellation correct.
- **B3 + all 57 greens byte-identical:** perpendicular caps route to `farCrossSectionArc` verbatim
  (call-graph identity); the Arc3d bite path is verbatim; N7's canal far-runout untouched.
- **Zero tuned constants; oracle-gated; honest-reject; shared-edge identity; analytic-on-arm trims.**
- **Before any PR:** full suite + lint + coverage; live MCP-bridge sphere-fillet + screenshot. NO PR until
  the whole corpus is green.

## Risks & escalations
- **Sphere-host retrim (FR4, the S7 gap):** new coverage — may surface a weld obstruction (the degenerate
  pinch vertex, a host retrim that won't close). Honest-report the exact gap; escalate genuinely new
  geometry.
- **N7 canal far-runout strangler:** deferred (ADR-1) — do NOT refactor N7 onto the engine in this plan;
  a later slice, behind its own byte-diff gate.
- **The un-exercised port pairings** (Torus∩Sphere, ∩Cone/∩Cyl): ship validated but untested-in-integration
  until the cone/torus families exercise them; a follow-on slice gates them per-family.
- **Tessellation (FR5):** apply the N7 trimmed-sub-span lesson proactively.

## References
- `.superpowers/sdd/far-runout-engine-architecture.md` (ADR-1..4, the seam, byte-identity contract).
- `.superpowers/sdd/far-runout-port-math.md` (the port math: pairings, feet, branch selection, DRAWEXE).
- `.superpowers/sdd/sphere-host-corner-derivation.md` (SP1/SP2 + the S7-gap retrim flag).
- `docs/superpowers/plans/2026-07-18-sphere-host-corner.md` (SP1–SP4, SP1/SP2 landed) + the N7 far-runout
  arc (the arm→corner→weld→far-runout methodology, the tessellation lesson).
- Patrikalakis & Maekawa (SSI/torus sections); Glaeser & Stachel (torus section catalogue); OCCT ChFi3d.
