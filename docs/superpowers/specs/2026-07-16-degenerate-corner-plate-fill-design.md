<!-- SPDX-License-Identifier: GPL-2.0-only -->

# M6 — Degenerate-corner variational plate fill (green N7 honestly)

## Status & provenance

The N7 arc established (all evidence-grounded, DRAWEXE + OCCT-source):
- `.superpowers/sdd/n7-runout-rederivation.md` — N7 is a tangent-degenerate trihedral corner; its face
  `result_5` is a rational BSpline, not a sphere octant.
- `.superpowers/sdd/n7-fill-rails-rederivation.md` — the corner's 4 rails are reflected-family
  cross-section circles (E1/E3/E4) + an on-wall bridge (E2); these rails are CORRECT and reproduce the
  oracle vertices exactly. **They are kept.**
- `.superpowers/sdd/n7-plate-solve-derivation.md` — OCCT's analytic 3-corner path **bails** at this
  degeneracy (`PerformThreeCorner` → `PerformMoreThreeCorner`) and builds a **`GeomPlate`** surface
  (Duchon order-3 polyharmonic plate, RBF r⁴log r + quadratic reproduction, dense saddle solve, then
  BSpline approx). This is why `result_5` is a BSpline.
- `.superpowers/sdd/n7-plate-seam-architecture.md` — the plate is a **new `plateProvider` tier** in the
  RailLoop engine, disambiguated from `coons4` by an extractor-stamped marker; it **complements** coons4
  (never replaces it); B3 stays byte-identical.

**What this supersedes:** the tuned-constant E2 (`wallBridgeFullness = 1.136`, commit `8e4d6207`) is
REJECTED — it made the per-face area gate circular (user decision: port the exact plate solve, no tuned
scalars). The reflected-family RAILS from `8e4d6207` are kept; the fill INTERIOR + E2 + the circular
gate are replaced by this milestone. Base for M6: `8e4d6207` (corpus 55).

## Goal

Green OCCT `tests/blend/simple/N7` (whole-body **61222.9**; corner `result_5` **90.194**) by adding a
variational Duchon plate corner-fill to the RailLoop engine, so the degenerate-corner fill is a faithful
port of OCCT's `GeomPlate` — **area emergent, no tuned constant.** Keep B3 byte-identical and the M1–M4
corpus unchanged. Corpus 55→**56**.

## Architecture (from the seam brief — ADR-1/2/3)

Ports-&-adapters, mirroring `coons4`:
- **`kernel/geom`** owns the math (imports only `math`): the Duchon solver + BSpline finish
  (`plate_average_plane.go`, `plate_solve.go`, `plate_fill.go`). Reuses `gaussSolve`
  (`fitted_bspline.go`), `ApproximateSurfaceLS` (`approximate_surface.go`), `MatchSurface`.
- **`kernel/ops`** owns the thin, topo-free `plateProvider` (`corner_provider_plate.go`):
  `Name()=BlendKindPlate`, `Fits`, `Build` → `geom.PlateFill` → `Certificate` (reuse coons4/sphere
  helpers) → `CornerBlendPatch{Kind: BlendKindPlate}`.
- **Tier order (the classifier):** `blendTiers() = [analyticSphere, plateProvider, coons4, tri3]`.
  Plate **complements** coons4: both Fit N7; plate wins by order; a plate honest-reject falls through to
  coons4 (N7 downside-protected — plate can only improve N7 or fall back to today's coons4).
- **Disambiguation (ADR-3, the non-regression safety):** a marker `RailLoop.Signature`
  (`RailSignatureGeneral=0 | RailSignatureTangentPlate`), stamped by **exactly one** function
  (`extractTangentDegenerateCorner`, after `wallFeetSplit` confirms the degeneracy).
  `plateProvider.Fits = (Signature==RailSignatureTangentPlate && Valence()==4)`. **Purely geometric
  `Fits` is rejected** — the Certificate admits ANY valid fair surface, so a geometric false-positive on
  a non-N7 valence-4 loop would let plate win by tier order and silently change that corner. The marker
  is CLASSIFICATION (a plain enum; `geom` never reads it), not geometry.

**Non-regression (airtight):** (1) `analyticSphere` is still tier 0 → wins the clean octant (valence-3,
unmarked) → B3 byte-identical, plate never invoked. (2) Only the tangent extractor stamps the marker;
every other extractor leaves `Signature==0` → `plateProvider.Fits` false → coons4 reaches every existing
case unchanged. Net delta = exactly one behaviour: a marked N7 loop whose plate cert passes uses plate.

**No public API** — ADR-0018 not triggered; all kernel-internal (`BlendKindPlate` is a `kernel/ops`
telemetry const). `Certificate` needs no new field (area is a P5 test-time oracle, not a runtime field).

## The math to port (from the plate-solve derivation — the authoritative spec for P1–P3)

Order m=3 Duchon plate: minimise ∫_Ω Σ C(3,k)(∂³f/∂u^k∂v^{3-k})² over a 2D average-plane domain Ω, per
coordinate. Minimiser = polyharmonic surface spline: `f(x) = Σ λ_i ∂^(Idu_i,Idv_i)E_m(x−P2d_i) + Σ a_α
x^α`, RBF `E_m(P)=R²logR` (R=r²=|P|², i.e. r⁴log(r²)), quadratic reproduction basis {1,u,v,u²,uv,v²}
(dim 6). Bordered saddle system per coord (X/Y/Z): `[K Pᵀ; P 0][λ;a]=[v;0]`, `K_ij = signe·E_m^(iu,iv)`,
solved by dense Gauss + one-step iterative refinement (reuse `gaussSolve`; reject non-`IsDone`).
Constraints: each rail sampled → G0 pinpoint rows (0,0); G1 rows (1,0)/(0,1) carrying the neighbour
surface tangents. Plate→BSpline: grid-eval + `ApproximateSurfaceLS` (degmax≤8, ~9 spans) + optional
`MatchSurface` to nail G1 < res.Weld.

### E2 — prescribe on-wall, no free scalar (the honest area)

`result_5` is G1 to **four** neighbours including the wall (a shared exact edge, not an open rim).
**Prescribe E2** on the wall as the reflected-family bridge (ball centre on the R−r=45 inner-offset
cylinder, azimuth C→C″, z by a **STANDARD cubic Hermite** whose end-slope DIRECTIONS match E4@V3 and
E1@V0 and whose handle MAGNITUDE is the standard chord-based value — **no tuned "fullness" scalar**),
radially projected to R=50. Feed all 4 rails as G1 plate constraints; the plate solves only the
interior; **the area is a pure functional of {4 fixed rails, m=3 energy} — emergent, no constant fit to
it.** Watertight by construction (E2 is an exact shared edge with the retrimmed wall).

**Fallback (OCCT-literal, if prescribed-E2 area misses tol):** emerge-E2 — constrain only the 3 arm
rails G1, leave the wall side as a plate contour, solve the fair interior, then intersect the plate with
the wall + same-parameter to imprint the exact wall edge. Exact OCCT area; adds a plate↔wall SSI +
same-parameter step. P5 decides: if prescribe-E2 greens the area gate honestly, ship it; else escalate
to emerge-E2.

## Tasks (7; math-advisor owns P1–P3, implementer owns P0/P4a/P4b/P5)

Order: **P0 → P1 → P2 → P3 → P4a → P4b → P5.** Each keeps the engine topo-free, reduces to B3 (plate
never runs on the octant), corpus stays 55 until P5.

- **P0 — marker + stamp + Fits (ops, ~15 LOC).** Add `RailLoop.Signature` field + `RailSignature`
  enum (`RailSignatureGeneral=0`, `RailSignatureTangentPlate`); stamp it in
  `extractTangentDegenerateCorner` after `wallFeetSplit`; add `plateProvider` STUB with
  `Fits = marker && valence==4` (Build returns false for now) and insert it in `blendTiers()` between
  analyticSphere and coons4. Gate: corpus 55 byte-identical (stub always declines → coons4 still serves
  everything); B3 unchanged.
- **P1 — average-plane Ω (geom, ~100 LOC).** `plate_average_plane.go`: the 2D parameter domain (an
  average plane through the corner region) + project the constraints into it. Test: projected
  constraints distinct, plane well-conditioned.
- **P2 — Duchon TPS core (geom, ~300–400 LOC).** `plate_solve.go`: `geom.PlateSolve(...)` — the E_m
  kernel + derivative rows + quadratic border + `gaussSolve` + one-step refinement; reject non-done.
  Unit-test vs an analytic minimiser (a known polyharmonic function reproduced to tol).
- **P3 — constraint discretization (geom, ~150 LOC).** `plate_fill.go`/`plate_constraints.go`: sample
  each `Side.Curve`→G0 rows, `Side.Adjacent`→G1 tangent rows; E2 is already the 4th Side. Test: the
  assembled constraint set matches the 4 rails + G1.
- **P4a — plate→BSpline finish (geom, ~60 LOC).** grid-eval the plate + `ApproximateSurfaceLS`
  (degmax≤8, ~9 spans) + optional `MatchSurface`; `geom.PlateFill(rails, adj, cont, tol)
  (BSplineSurface, error)` — error ⇒ caller falls through. Test: fills the N7 rails, G0/G1 residual < tol.
- **P4b — provider + tier (ops, ~100 LOC).** `corner_provider_plate.go`: `Build` maps `Sides` →
  `PlateFill` → certify (reuse) → `CornerBlendPatch{Kind: BlendKindPlate}`; honest-reject ⇒ fall through
  to coons4. Test: N7's marked loop → `BlendKindPlate`; a non-marked valence-4 loop → coons4 (unchanged).
- **P5 — DRAWEXE gate + green N7 (tests).** Whole-body Σ=61222.9 + per-face (result_5=90.194
  **emergent**, not tuned) + watertight + volume + E2-on-wall witness (`|dist(axis)−R|≤res.Weld·R`) +
  G1-to-wall along E2 + corpus 55→**56** + B3/M4 byte-identical. If prescribe-E2 misses the area gate,
  escalate to emerge-E2 (do NOT tune a scalar).

## Verification & gates
- **N7 whole-body** 61222.9 + **per-face** result_5=90.194 **emergent** (a functional of the fixed rails
  + energy; NO free scalar — this is the whole point of M6), all 12 faces, watertight + volume; corpus
  55→56.
- **B3 byte-faithful** — octant valence-3 unmarked → analyticSphere wins, plate never runs; B3 golden +
  weld + volume + corpus subtest unchanged.
- **M1–M4 tripwire + whole corpus byte-identical** except N7; the marker defaults zero so no other
  extractor/case changes; `coons4`/`analyticSphere`/`tri3` behaviour unchanged.
- **Honest gate** — the plate path carries NO tuned scalar; area only CHECKS the solve. E2 on-wall +
  G1-to-wall witnessed independently of area (coons4's ParamAt re-projection trap does not apply — the
  plate uses the analytic rail).
- **DRAWEXE oracle** — the N7 recipe (import `simple/N7.step`, blend edges at r=5; the vendored
  `CFI_f1234fim.rle` is the WRONG shape — reconcile it) confirms 90.194 + Σ + volume.
- **Engine dependency rules** — `geom/plate_*.go` import only `math`; `ops/corner_provider_plate.go`
  import only geom+math; no `topo` in either; the marker is the sole extractor→provider channel.

## Risks & out-of-scope
- **RBF conditioning** (the one real numerical risk) — OCCT ridge-guards; a discrete-biharmonic FEM is a
  well-conditioned equivalent if the dense RBF solve is ill-conditioned at N7's ~40–120 constraints.
  Honest-reject on a non-converged solve → coons4 fallback.
- **prescribe-E2 vs emerge-E2** — prescribe-E2 is primary (watertight, no scalar); emerge-E2 (SSI +
  same-parameter) is the exact fallback if the prescribed area misses tol. Decided at P5, never by tuning.
- **Fixture reconciliation** — fix the mislabeled vendored `CFI_f1234fim.rle` (a different 8-face shape)
  so the manual DRAWEXE oracle recipe matches `simple/N7.step` (the corpus fixture).
- **Concave/other degenerate corners** — M6 targets the tangent-degenerate trihedral (N7) class; other
  degenerate families reuse the plate tier as they arise (the marker + tier generalize).
- **ADR-2 Step-2 octant loop-collapse** (from the T-N7.0 seam) — still a separate follow-up.

## References
- `.superpowers/sdd/n7-plate-seam-architecture.md` (ADR-1/2/3, the seam contract, task shape).
- `.superpowers/sdd/n7-plate-solve-derivation.md` (the Duchon math, prescribe-vs-emerge E2, honest area).
- `.superpowers/sdd/n7-fill-rails-rederivation.md` (the reflected-family rails, kept).
- In-repo ADR-0051 (provider tiers + honest-reject + lineage invariance), ADR-0042 (model-relative
  tolerances), ADR-0018 (public-API split — not triggered).
- Duchon (1977) surface splines; Wahba, *Spline Models for Observational Data* (1990); OCCT
  `Plate_Plate`/`GeomPlate_BuildPlateSurface`/`GeomPlate_MakeApproxSurface` (reference implementation).
