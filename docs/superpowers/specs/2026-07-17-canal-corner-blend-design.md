<!-- SPDX-License-Identifier: GPL-2.0-only -->

# M6′ — Rolling-ball canal corner blend (green N7 faithfully)

## Status & provenance — why this supersedes the plate

The M6 plate milestone rested on the inference that OCCT's degenerate-corner face `result_5` (area
90.194) is a variational (Duchon) plate. **That inference was empirically overturned.** Evidence chain
(all DRAWEXE- or spike-verified):

- **DRAWEXE oracle re-confirm** (`test-utilities/occt-blend/data/CFI_f1234fim.rle`, `tscale 10`, blend
  `s_5 s_4 s_10` @ r=5): whole-body 61222.9, `result_5` = **90.194**, 4-sided, R=50 caps. The vendored
  fixture is correct (the prior "8-face/8.04" note was a misread). Pole net saved at
  `.superpowers/sdd/result5-poles.txt`.
- **`result_5` is a rolling-ball CANAL surface, not a plate.** Its dump is BSplineSurface, Degrees **2×9**,
  Poles **3×10**, u-rational with middle-row weights ≈ **cos 45°** → every u-cross-section is an exact
  **radius-5 circular arc** (the rolling ball). The `v=0` isocurve is a radius-5 quarter-circle centered
  at **C″=(55,5.279,5)** — a reflected-family ball center.
- **Spike 1 (`experiments/n7-coupled-spike`)** proved a bending-minimal / free-magnitude-tangency plate
  on the 4 rails reaches only **51** area with **exact** G1 (0.0002) — so 90.194 is *not* a plate result.
- **Spike 2 (`experiments/n7-blend-sweep`)** reconstructed `result_5` from our corner-engine data as a
  canal sweep to **−0.025% area (90.1716)** and **0.005 shape deviation**, with the spine as the
  **inner-offset host SSI** (a rail-based spine missed by +75%).

This vindicates the standing "do like OCCT ChFi3d" oracle: `result_5` is a rolling-ball blend patch. The
landed Duchon plate solver (`kernel/geom/plate_*.go`, P0–P4a, P2.1 — all reviewed, green) is **correct
code for a genuinely-variational corner but the wrong model for this face**; it is kept dormant, not the
build target here. Base for M6′: `02fd1932` (corpus 55).

## Goal

Green OCCT `tests/blend/simple/N7` (`result_5` = **90.194**, whole-body **61222.9**) by adding a
rolling-ball **canal-surface corner provider** to the RailLoop engine, faithful to OCCT ChFi3d — area
emergent from the rolling-ball geometry, **no tuned constant**. Keep B3 byte-identical, M1–M4 unchanged.
Corpus 55→**56**.

## The construction (from the two consults — the authoritative math + seam)

Math: `.superpowers/sdd/canal-corner-math.md`. Seam: `.superpowers/sdd/canal-corner-seam-architecture.md`.

**Surface = envelope of a radius-r ball rolling through the corner** (a canal patch): radius-r circular
cross-sections swept along the ball-center spine.

1. **Spine = inner-offset host SSI.** `m(v) ∈ H_a^{−r} ∩ H_b^{−r}` — offset each of the corner's TWO roll
   hosts by r **into the cavity** (sign = material side: N7 wall 50→45, s_10 5→10), then intersect the
   OFFSETS (never the raw hosts, never the rim rails — the rail spine missed +75%). Closed-form offsets:
   plane→parallel plane, cylinder ρ→ρ∓r, cone→apex-slid same-α, torus→r_min∓r; elliptical-cylinder →
   march. Closed-form SPINE for plane∩{plane,cyl,cone,sphere}, coaxial pairs, and the orthogonal/
   equal-radius cylinder specials (N7's case) via `geom.IntersectSurfacesAnalytic`; MARCHED (general
   cyl∩cyl, anything∩torus/elliptical-cyl) via `geom.TraceSurfaceIntersection`. v-endpoints = the
   reflected-family centers C/C′/C″ on that pair's spine (N7: C″→C).
2. **Cross-section arc at m(v):** the exact radius-r rational-quadratic arc `f_a → shoulder → f_b`, weight
   `cos(½ ∠(f_a, m, f_b))`, in the plane of the two feet + center, on the cavity side. Feet `f_a, f_b` =
   the ball's two host-contact points (at distance r from m by construction; the two-feet plane IS the
   spine normal plane).
3. **Parametrization — u = arc parameter, v = chord-length along the spine** (the watertight crux, and
   where our canal BEATS OCCT's deg-9 diagonal). This makes ALL FOUR rails **isoparms**: v-ends = the two
   neighbour-arm end arcs, u-ends = the two foot-loci. So the provider **shares the exact edge CURVES**
   with its four neighbours (watertight by construction — welding needs shared edge objects, not isoparam
   match). Each foot-locus is kept provably ON its host (`|dist| ≤ res.Weld·scale`). We do NOT chase
   OCCT's 0.557 iso-line gap — that was pure parametrization; the SHAPE matches to 0.005.
4. **BSpline emission = exact homogeneous loft** (Piegl & Tiller §10.3) of the three arc control-curves
   `(w·x, w·y, w·z, w)` → degree-2-rational-in-u × spline-in-v, 3×N (OCCT's form; middle pole-row weight
   = cos β(v)). Exact when the spine is closed-form (low-degree rational rows), interpolating loft when
   marched; least-squares only for noisy marches, never OCCT's degree 9. Reuse `geom.NewBSplineSurface` /
   `ApproximateSurfaceLS`, `NewConicSectionCurve` / `NewRuledSectionBlend`.
5. **B3 reduction (keeps the clean octant a sphere):** clean octant → the three host-offsets concur →
   spine length → 0 → the canal degenerates to the sphere of radius r about the concurrence center →
   `analyticSphere` wins byte-identical. Predicate: spine length `≤ res.Weld·scale` ⇒ decline (guard
   against emitting a rank-deficient point-spine loft). N7's degenerate valence-4 corner has a genuine
   (non-zero) spine C″→C → canal builds.

## Architecture (from the seam brief — minimal, mirrors the plate/obstacle seam)

- **New request payload (the minimal RailLoop addition):** `RailLoop.Canal *CanalCorner` where
  `CanalCorner struct { Rolls []geom.Surface; Radius float64 }` — nilable, geom.Surface-only (topo-free,
  ADR-0051), mirroring the existing `CornerBlendRequest.ObstacleFeature` optional-payload pattern.
  `Side.Adjacent` is insufficient (it carries the ARM surfaces coons4's G1 ribbon needs, and the second
  roll host is absent), so the two roll hosts + explicit r ride the payload. Populated by
  `extractTangentDegenerateCorner` (it already resolves both hosts and knows `w.radius`); r is explicit
  (reading ±r off a rational arc is fragile).
- **`canalProvider` (`kernel/ops/corner_provider_canal.go`, topo-free, geom+math only):**
  `Name()=BlendKindCanal`; `Fits(l) = (l.Canal != nil && l.Valence()==4)` — the payload pointer IS the
  extractor stamp (stronger than an enum: it also guarantees the spine data is present); `Build` →
  `geom.CanalCornerFill(rolls, radius, rails, res)` → certify (reuse coons4/sphere `Certificate`) →
  `CornerBlendPatch{Surface, Loops: railLoopToFilletLoops(loop), Kind: BlendKindCanal}`; honest-reject
  (offset self-intersects, SSI non-converged, point-spine) → falls through to coons4.
- **Tier order:** `blendTiers() = [analyticSphere, canalProvider, coons4, tri3]` — canal takes the plate
  stub's slot. Clean octant (valence-3, `Canal=nil`) declined upstream by the extractor → analyticSphere
  wins → **B3 byte-identical**. Every non-N7 case keeps `Canal=nil` → canal declines → coons4/sphere/tri3
  unchanged.
- **plateProvider disposition:** REMOVE the `kernel/ops` plate stub (a Build-always-declines no-op) + its
  test + the `RailSignatureTangentPlate` marker; **KEEP `kernel/geom/plate_*.go` dormant** (complete,
  reviewed Duchon solver — correct for future genuinely-variational corners; deleting tested code is the
  irreversible move).
- **Weld stays agnostic:** the assembly/orient/weld layer just consumes surface+loops. ONE required edit:
  `curvedCornerFace`'s Kind-gate whitelist (`fillet_curved_weld.go:~340`) — add `case BlendKindCanal`
  (else the patch is silently dropped). Watertight constraint: `Build` MUST emit `Loops` on the RECEIVED
  rails (`railLoopToFilletLoops(loop)`), exactly as coons4 does.

## Seam signatures (the handoff contract)
- `kernel/geom`: `CanalCornerFill(rolls []geom.Surface, radius float64, rails [4]geom.Curve3, res Resolution) (BSplineSurface, error)` — the spine-SSI + cross-section-loft; imports only `math`. Plus internal
  `canalSpine`, `crossSectionArc`, `loftCanal` helpers.
- `kernel/ops`: `RailLoop.Canal *CanalCorner`; `CanalCorner{Rolls []geom.Surface; Radius float64}`;
  `BlendKindCanal CornerBlendKind`; `canalProvider` (Name/Fits/Build).

## Tasks (sub-milestone; SDD; math-owned geom vs mechanical ops)
Order is load-bearing — geom math bottom-up first, then ops wiring, then weld+green:
- **C0 — RailLoop.Canal payload + extractor populate + plate-stub removal (ops, ~40).** Add the payload +
  `BlendKindCanal`; populate in `extractTangentDegenerateCorner` (both roll hosts + r); remove the plate
  ops stub + marker; insert `canalProvider` STUB (Build declines) in `blendTiers()`. Gate: corpus 55
  byte-identical (stub declines → coons4 still serves N7); B3 unchanged.
- **C1 — offset-SSI spine (geom, ~150).** `canalSpine(rolls, radius, res)` → offset each host + SSI
  (analytic dispatch → marched fallback), endpoints at the reflected-family centers; math-advisor owns the
  offset/SSI dispatch + the tangential-pinch Gauss-Newton guard. Test: N7 spine reproduces C″→C, radius-5
  feet, vs the spike.
- **C2 — cross-section arc + homogeneous loft (geom, ~150).** `crossSectionArc(m, f_a, f_b, r)` (exact
  rational-quadratic) + `loftCanal` (Piegl&Tiller §10.3 homogeneous loft, u=arc/v=chord-length). Test: the
  N7 canal area = 90.194 within `res.Weld·r²` (EMERGENT), 4 rails as isoparms, feet on-host.
- **C3 — `CanalCornerFill` + `canalProvider.Build` + tier (geom+ops, ~120).** Compose C1+C2; provider
  Build → certify → `CornerBlendPatch{Kind:BlendKindCanal}`; honest-reject paths. Add `BlendKindCanal` to
  the `curvedCornerFace` Kind-gate. Test: N7 marked loop → BlendKindCanal area 90.194; B3 → sphere;
  non-marked → unchanged.
- **C4 — weld T-N7.3/T-N7.4 + green N7 (ops+tests, ~150).** Wall termination + far-runout weld the canal
  patch (reuse the corner-fill plan's bittenLoop / far-runout, unchanged by the surface model); whole-body
  Σ=61222.9 + per-face 90.194 + watertight + volume + corpus 55→**56**.
- **C5 — DRAWEXE gate + non-regression (tests).** Oracle re-confirm; M1–M4 byte-identical; B3 golden/weld/
  volume; full suite + lint + coverage>80%/dup<3%.

## Verification & gates
- **N7:** whole-body 61222.9 + `result_5` 90.194 **emergent** (from the rolling-ball geometry — NO tuned
  scalar), all 12 faces watertight + volume; corpus 55→56; DRAWEXE-confirmed.
- **B3 byte-faithful:** clean octant → `Canal=nil` → analyticSphere → plate/canal never run; B3 golden/
  weld/volume + corpus subtest unchanged. The canal's own B3 math reduction (spine→0→sphere) is a
  belt-and-suspenders check, not the routing path (routing is the nil payload).
- **Whole corpus byte-identical except N7:** the payload is nil everywhere else; coons4/analyticSphere/
  tri3 unchanged; the plate ops-stub removal is corpus-neutral (it always declined).
- **Honest gate:** NO tuned scalar on the canal path; the area gate CHECKS the rolling-ball geometry, it
  does not fit it. Feet-on-host + shared-edge-curves witnessed independently of area (watertightness).
- **Engine purity:** `geom/canal_*.go` import only `math`; `ops/corner_provider_canal.go` import geom+math,
  no `topo`; the `Canal` payload is the sole extractor→provider channel.

## Risks & out-of-scope
- **Offset-SSI tangential ill-conditioning at the corner** (the corner IS a degeneracy): detect
  `|n_a×n_b| < ε_tan`, seed the SSI at the exact reflected-family center endpoints, switch Newton →
  Gauss-Newton on ‖offset-residual‖² through the pinch. Honest-reject on non-convergence → coons4.
- **Offset self-intersection** when r > host min-curvature (`OffsetSurface.SelfIntersects`) → honest-reject.
- **Marched-spine cases** (general cyl∩cyl, torus, elliptical-cyl) beyond N7 are supported by the
  `TraceSurfaceIntersection` fallback but exercised as they arise in the corpus; N7 is a closed-form
  (cylinder-specials) spine.
- **plate_*.go dormancy:** kept, untested-in-integration but unit-tested; a future genuinely-variational
  corner reactivates it. Not deleted.
- **ADR-2 octant loop-collapse** (from the T-N7.0 seam) remains a separate follow-up.

## References
- `.superpowers/sdd/canal-corner-math.md` (spine-SSI, cross-section, u/v parametrization, loft, B3, pitfalls).
- `.superpowers/sdd/canal-corner-seam-architecture.md` (the RailLoop.Canal payload, tier, plate disposition, weld).
- `.superpowers/sdd/blend-sweep-spike-report.md` (the −0.025% reconstruction + host-offset spine).
- `.superpowers/sdd/n7-fill-rails-rederivation.md` (rails + reflected-family centers C/C′/C″).
- `.superpowers/sdd/result5-poles.txt` (OCCT's exact 3×10 net — the oracle for the emission test).
- In-repo ADR-0051 (provider tiers), ADR-0042 (relative tolerances), ADR-0018 (public-API split — not triggered).
- Patrikalakis & Maekawa, *Shape Interrogation for CAD/CAM* (SSI/canal/offsets); Maekawa (1999, offsets);
  Rossignac & Requicha (1984, blends); Pottmann & Peternell, *Computational Line Geometry* (canal
  surfaces); Piegl & Tiller, *The NURBS Book* §10.3 (rational loft); OCCT ChFi3d (reference implementation).
