<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Sphere-host trihedral corner (curved-host campaign, slice 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use
> checkbox (`- [ ]`) syntax. Implementers run on OPUS; reviewers on FABLE (user directive: bulletproof,
> no compromise, same rigor as N7).

**Goal:** Green OCCT `tests/blend/simple/{D5, D9, E4}` — the three SPHERE-host trihedral corners — by
teaching the fillet kernel two exact-analytic pieces it lacks: the **Sphere∧Plane arm fillet** (an exact
torus) and the **sphere-host corner solve** (an analytic sphere tangent to the host sphere + 2 planes),
then welding the whole body. Corpus 57→**60**. This opens the trihedral curved-host-corner campaign
(20 cases: sphere 3 → cone 4 → torus 4 → the 8 others).

**Architecture:** Extend the EXISTING analytic corner path (not the RailLoop engine): `solveBlend`'s
curved-host branch (`cylinderHostCorner`→`solveCurvedBlend`, the M5 Slice-A path) generalizes to a
`curvedHostCorner` recognizer (one curved host + 2 planes) whose sphere case solves the ball centre on
the plane-pair line at `|c − O| = R − r` (convex). The Sphere∧Plane arm is an exact torus (spine =
offset-plane ∩ ρ-sphere circle) added to `computeEdgeFillet` alongside `cylinderPlaneEdge`. Everything is
exact; no tuned constants. Reduces to the cylinder (M5) + all-planar corner paths byte-identical.

**Tech Stack:** Go (`oblikovati` GPL, `kernel/ops` + minimal `kernel/geom`). Reuse `planePairLine`,
`selectCornerRoot`, `curvedCornerConsistent`, `curvedCornerTangents`, `geom.NewTorus`, `geom.NewSphere`,
the weld/assembly machinery. DRAWEXE oracle.

**Base:** branch tip after the N7 milestone (`3e9a7dc7`), corpus 57. Derivation (authoritative math):
`.superpowers/sdd/sphere-host-corner-derivation.md`.

## Global Constraints

- **NO PR until the whole corpus is green** (this slice targets 57→60; the blend corpus is far from
  fully green — accumulate + commit per task, no PR).
- **All 57 existing corpus greens BYTE-IDENTICAL** (incl. N7/B3/L3/L6): the sphere recognizer is ordered
  AFTER the untouched `cylinderHostCorner` and before `solvePlanarBlend`; the cylinder (M5) + all-planar
  paths stay unreachable-by-construction for non-sphere hosts. Gate every task on the full
  `TestOCCTBlendSimple` verdict set byte-identical to base except the target cases.
- **Zero tuned constants.** Every centre/radius/tangency is a closed form of `{r, R, ρ, plane normals}`;
  the area gates CHECK against the OCCT oracle, never fit. `res.Weld·scale` tolerances (ADR-0042), no
  bare 1e-6 (scale-free angular guards must be justified dimensionless).
- **Honest reject, never a forced green.** A sphere-host config that has no valid equal-r ball (spindle
  R ≤ r, concave bore R+r out of this slice, grazing tangency, reflected-root ambiguity) returns "corner
  face must be planar" / the exact existing reject — do-no-harm. NEVER loosen a gate; NEVER ship a
  mis-closed shell. If the whole-body weld won't close, report the exact obstruction (like N7's slices).
- **Oracle-gated:** derived arm/corner geometry matches DRAWEXE-measured faces (D5 corner centre residual
  3.7e-10, arm ≤2.3e-4 — the derivation's numbers); whole-body Σ + per-face areas gate against the corpus
  oracle (D5 134780, D9/E4 per corpus.json). The whole-body gate is `TestOCCTBlendSimple/{D5,D9,E4}` on
  the REAL STEP body.
- **Functions 4–20 lines (golangci funlen 30/20); files < 500; SPDX GPL-2.0-only on new files; explicit
  types; early returns; ≤2 indent; errors carry offending value + expected shape.** `math.NewPoint3` does
  NOT exist → `math.P3`.
- **Corpus count (the `-v` is REQUIRED):**
  `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'`
- **DRAWEXE oracle:** `source test-utilities/occt-blend/oracle/drawenv.sh`; D5 recipe `psphere s 15 -60 60
  90; trotate s 0 0 0 0 0 1 90; tscale s 0 0 0 10; explode s E; blend result s 10 s_2 10 s_1 10 s_6`.

## The oracle-verified geometry (from the derivation — the spec for SP1/SP2)

- **Sphere∧Plane arm = exact torus.** Filleting an edge where a plane meets the host sphere (R): the
  rolling-ball fillet surface is a torus whose spine is the circle `offset-plane(∓r) ∩ sphere(R∓r)` (a
  planar circle), minor radius r. Exact in every configuration (no oblique hole). Oracle arm samples match
  the derived tori to ≤2.3e-4 (OCCT's own approximation tolerance). Plane∧Plane third edges keep the
  existing `cylinderPlaneEdge` path.
- **Sphere-host corner = analytic sphere.** The two planes pin `c` to the plane-pair line
  (`planePairLine`, point p₀ r-inside both, direction d = n̂₁×n̂₂). The host-sphere tangency is the full-3D
  quadratic in the line parameter t: `qa=|d|², qb=2u·d, qc=|u|²−ρ²` with `u = p₀ − O`, **ρ = R − r for the
  CONVEX case (D5: (v−O)·n̂_out > 0, material inside the host sphere)**; concave bore (R+r) is out of this
  slice (honest-reject). Roots → `selectCornerRoot` keeps the material-outward `c₀` (no cylinder arms to
  witness; oracle-correct in all 3). DRAWEXE-verified centres: D5 (−71.5757, 10, 119.9038) res 3.7e-10;
  D9 (−10, −71.5757, 119.9038) 3.7e-10; E4 (−10, 10, −139.2839) 4.8e-12. Corner trim = spherical triangle
  of the 3 arm-rail great-circle arcs + the degenerate host-tangency vertex; area 55.7891 (D5) matches OCCT.
- **Reduces to existing paths byte-identical** (cylinder M5 + all-planar). Flagged NEW coverage risk:
  the sphere-host **retrim/tessellation** is untested (`setbackBossesFaithful` has no Sphere — the S7 gap);
  SP3 is where this surfaces, handled with the same honest-report discipline as N7's weld slices.

## File Structure
- `kernel/ops/fillet_sphere_arm.go` (create) — the Sphere∧Plane torus arm (spine circle + `geom.NewTorus`).
- `kernel/ops/fillet.go` (modify) — `computeEdgeFillet` dispatch: add the Sphere∧Plane case beside
  `cylinderPlaneEdge`; `solveBlend`→`curvedHostCorner`.
- `kernel/ops/fillet_curved_corner.go` (modify) — `curvedHostCorner` (generalize `cylinderHostCorner` to
  one-curved-host+2-planes), `sphereHostCornerCenter` (the quadratic), the sphere branch of
  `solveCurvedBlend`.
- `kernel/geom` — only if a plane∩sphere circle / offset helper is missing (grep + reuse first).
- `model/feature/occt_blend_simple_test.go` / a sphere-corner test — SP3 whole-body + per-face gates.
- Tests alongside each.

Task order — **SP1 → SP2 → SP3 → SP4.** SP1 (torus arm) makes the arms build; SP2 (corner) makes the
corner solve; SP3 assembles + greens D5/D9/E4; SP4 gates. Corpus stays 57 until SP3.

---

### Task SP1: Sphere∧Plane arm fillet (exact torus)

**Files:** Create `kernel/ops/fillet_sphere_arm.go` (+ `_test.go`); modify `fillet.go` (`computeEdgeFillet`
dispatch).

**Interfaces:**
- Consumes: `computeEdgeFillet` (read how `cylinderPlaneEdge` builds a cylinder∧plane arm — the sibling
  to mirror), `geom.NewTorus`, plane∩sphere circle geometry, `edgeFillet`/`armSurface`, `res.Weld`.
- Produces: the Sphere∧Plane branch — recognize an edge whose two faces are {sphere host, plane}; build
  the torus arm: spine = the circle `sphere(R∓r) ∩ plane(∓r offset)` (centre, axis, radius from the
  derivation), minor radius r; `armSurface = geom.NewTorus(...)`. Honest-reject the configs the derivation
  excludes.

- [ ] **Step 1 — Failing test (oracle-pinned):** on the REAL D5 body, the two Sphere∧Plane edges build
  torus arms whose surface matches the oracle's 2 rational-BSpline arm faces (type torus; sampled points
  ≤ the derivation's 2.3e-4; spine circle centre/axis/radius from the derivation). The Plane∧Plane edge
  still builds a cylinder arm. Assert `computeEdgeFillet` no longer errors on D5's sphere edges (the
  `curvedAdjacentError` is gone for these).
- [ ] **Step 2 — Run, verify it fails** (currently `computeEdgeFillet` only knows cylinder∧plane).
- [ ] **Step 3 — Implement** the torus arm + dispatch (helpers: recognize-sphere-plane-edge, spine-circle,
  build-torus, reject).
- [ ] **Step 4 — Run tests; verify pass.**
- [ ] **Step 5 — Corpus `57` byte-identical (D5/D9/E4 now fail at the CORNER, not the arm — verify the
  reason moved to "corner face must be planar"); B3/N7/all-57 unchanged; build/vet/gofmt/golangci-lint
  clean.**
- [ ] **Step 6 — Commit** (`feat(ops): Sphere∧Plane torus arm fillet (sphere-host campaign SP1)`).

---

### Task SP2: sphere-host corner solve (analytic sphere)

**Files:** Modify `kernel/ops/fillet_curved_corner.go` (`curvedHostCorner`, `sphereHostCornerCenter`,
`solveCurvedBlend` sphere branch); `fillet.go` (`solveBlend` dispatch). Test alongside.

**Interfaces:**
- Consumes: `cylinderHostCorner`/`solveCurvedBlend`/`curvedCornerCenter`/`planePairLine`/`selectCornerRoot`
  /`curvedCornerConsistent`/`curvedCornerTangents` (the M5 path to generalize), `geom.Sphere`/`geom.NewSphere`.
- Produces: `curvedHostCorner(faces) (host geom.Surface, planes [2]*topo.Face, kind, ok)` — extends
  `cylinderHostCorner` to {cylinder|sphere}+2-planes (cone/torus stubbed for follow-on slices);
  `sphereHostCornerCenter(sphere, planes, r, v, res) (math.Point3, bool)` — the plane-pair-line ∩ (R−r)
  offset-sphere quadratic (`qa/qb/qc` from the derivation) + `selectCornerRoot`; the sphere branch of
  `solveCurvedBlend` → `cornerBlend{center, sphere: NewSphere(c,r), tan: curvedCornerTangents}`.

- [ ] **Step 1 — Failing test:** on the REAL D5/D9/E4 bodies, the corner solve returns the analytic sphere
  corner: centre matches DRAWEXE (D5 (−71.5757,10,119.9038) to res.Weld·scale ~3.7e-10; D9/E4 per the
  derivation), radius r=10, tangent to the host sphere + 2 planes (`curvedCornerConsistent` passes). The
  "corner face must be planar" reject is gone for these. A concave-bore / spindle / grazing config →
  honest-reject (the exact existing string).
- [ ] **Step 2 — Run, verify it fails** (sphere host → `solvePlanarBlend` reject).
- [ ] **Step 3 — Implement** `curvedHostCorner` + `sphereHostCornerCenter` + the sphere branch (helpers:
  recognizer, offset-sphere quadratic, convex-sign guard, consistency).
- [ ] **Step 4 — Run tests; verify pass** (oracle centre residuals reported).
- [ ] **Step 5 — Corpus `57` byte-identical (D5/D9/E4 now build the corner + arms; whole-body may floor on
  the weld — SP3 — or green early; if it greens honestly with Σ→oracle, that's SP3's gate reached early);
  the cylinder (M5) + all-planar corners BYTE-IDENTICAL (the recognizer ordering); B3/N7 unchanged; lint
  clean.**
- [ ] **Step 6 — Commit** (`feat(ops): sphere-host analytic corner solve (sphere-host campaign SP2)`).

---

### Task SP3: whole-body weld + green D5/D9/E4 (corpus 57→60)

**Files:** Modify the weld/retrim as needed for the sphere host (the flagged new coverage); the corpus
test (per-face + whole-body gates). Honest-report any weld obstruction.

**Interfaces:** Consumes SP1 (torus arms) + SP2 (sphere corner) + the existing weld/assembly
(`assembleBody`, the retrim/far-runout machinery); the corner trim = the spherical-triangle rail loop.

- [ ] **Step 1 — Failing (whole-body) test:** `TestOCCTBlendSimple/{D5,D9,E4}`: whole-body Σ = the corpus
  oracle (D5 134780; D9/E4 per corpus.json) within `deps` tolerance, all oracle faces present, WATERTIGHT
  (Valid + HolesContained + IsSolid; every edge 2-incident), volume correct, per-face areas match the
  oracle (corner 55.7891 for D5, the torus arms, etc.).
- [ ] **Step 2 — Run, verify it fails** (weld not yet wired / the S7-gap retrim).
- [ ] **Step 3 — Implement** the sphere-host weld/retrim (reuse the existing machinery; extend
  `setbackBossesFaithful`-class coverage for Sphere ONLY if needed — the derivation flagged this as new).
  Honest-reject with the exact obstruction (which edge/face/gap) if a seam won't close — do NOT force.
- [ ] **Step 4 — Run; verify D5/D9/E4 green.** Corpus prints `60`.
- [ ] **Step 5 — Non-regression:** all 57 prior greens byte-identical; N7/B3/L3/L6 unchanged; full
  `go test ./kernel/... ./model/...` + lint clean.
- [ ] **Step 6 — Commit** (`feat(ops): green OCCT blend/simple/{D5,D9,E4} sphere-host corners (corpus 57→60)`).

---

### Task SP4: DRAWEXE gate + non-regression + tessellation check

- [ ] **Step 1 —** Re-run the DRAWEXE recipes for D5/D9/E4; confirm Σ + per-face + the corner-sphere
  centre/radius vs our build; record.
- [ ] **Step 2 —** Confirm the sphere-host faces TESSELLATE CORRECTLY (the torus arms + sphere corner mesh
  to their true areas, no folds — apply the N7 tessellation lesson: check the per-face mesh converges, not
  just the Σ). Fix any fold in the canal/sphere-arm sub-edge presentation (reuse `sampleCurve3OpenTrimmed`
  if applicable).
- [ ] **Step 3 —** Full `go test ./...` + `golangci-lint` + `gofmt -l` + markdownlint; whole corpus
  byte-identical except D5/D9/E4; coverage > 80% on new files.
- [ ] **Step 4 — Commit** (fold into SP3 if no new files).

---

## Verification
- **D5/D9/E4:** whole-body Σ = oracle, all faces, watertight, volume, per-face areas (corner analytic
  sphere, torus arms) — corpus 57→60; DRAWEXE-confirmed; tessellation correct (no folds).
- **All 57 prior greens byte-identical** (recognizer ordering + `Canal`/curved-host gating; the cylinder
  M5 + all-planar corner paths untouched).
- **Zero tuned constants; oracle-gated; honest-reject preserved; shared-edge identity for every seam.**
- **Before any PR:** full suite + lint + coverage; live MCP-bridge sphere-fillet + screenshot. NO PR until
  the whole corpus is green.

## Risks & escalations
- **Sphere-host weld/retrim (SP3, the S7 gap):** new coverage — may surface a weld obstruction like N7's
  arm-weld/far-runout. Honest-report the exact gap; escalate genuinely new geometry to the advisors.
- **Concave-bore sphere corners (R+r):** out of this slice (honest-reject); a follow-on if any corpus case
  needs it.
- **The follow-on cone/torus slices:** the derivation gives their tangency laws (cone = coaxial cone apex
  ∓r/sinα, quadratic; torus = offset torus, quartic) — the `curvedHostCorner` recognizer is structured to
  extend to them, but they are separate slices.
- **Tessellation (SP4):** apply the N7 lesson (trimmed sub-span geometry) proactively if the torus/sphere
  arm sub-edges present whole-curve geometry.

## References
- `.superpowers/sdd/sphere-host-corner-derivation.md` (THE math: torus arm, sphere corner quadratic,
  oracle residuals, `curvedHostCorner` shape, cone/torus laws, byte-identity note).
- `docs/superpowers/specs/2026-07-14-corner-extractor-wave-design.md` (the campaign roadmap).
- `docs/superpowers/plans/2026-07-17-canal-far-runout.md` + the N7 ledger (the arm→corner→weld methodology,
  tessellation lesson).
- OCCT ChFi3d (reference); the M5 `.superpowers/sdd/m5-curved-arm-derivation.md` (the cylinder-host path
  this generalizes).
