<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Cone-host trihedral corner — implementation plan (CN1–CN6)

> **For agentic workers:** execute via superpowers:subagent-driven-development — fresh implementer
> per task (opus), adversarial reviewer per task (fable), mesh-hash byte-identity gate. The exact
> geometry, formulas, DRAWEXE certificates, and numerical pitfalls live in
> `.superpowers/sdd/cone-host-corner-derivation.md` (the mathematician's derivation) — this plan is the
> task decomposition and the binding global constraints. Each task brief is written from the derivation.

**Goal:** green OCCT `blend/simple/{C2,C6,C8,D1}` (corpus 60→64) — the four CONE-host trihedral corners
(the "corner face must be planar" family) — with the mathematically-correct constant-radius rolling-ball
fillet, exercising the RailLoop/far-runout engine and adding the first non-analytic (canal BSpline) arm.

**Architecture:** the sphere-slice pattern (arm → corner → weld/retrim → far-runout → gates) plus the
canal-arm crux. One `geom.Cone` host + two `geom.Plane`s. Dispatch: `coneHostCorner` inserted AFTER
`sphereHostCorner`, before `solvePlanarBlend`; `coneArmEdge` after `sphereArmEdge`, before
`curvedAdjacentError` — so cylinder/sphere/planar paths are unreachable-by-construction (do-no-harm).

## Global Constraints (bind every task)

- **NO PR** — corpus far from whole-green; commit-on-branch only.
- **BYTE-IDENTITY of every prior green is a HARD gate, verified at the MESH-BIT level, NOT verdict-set.**
  D9-T2 lesson: a verdict-set (PASS/FAIL) diff is INSUFFICIENT when shared retrim/solve/orient code is
  touched — it missed a real E4 mesh drift. Each task's byte-identity gate compares, base worktree vs
  HEAD, an **ORDER-INDEPENDENT commutative triangle-bit fingerprint** of every prior-green body's full
  Property-quality tessellation — a mod-2⁶⁴ sum of per-triangle FNV-64a hashes. Do NOT use a POSITIONAL
  FNV over (positions+indices): `TessellateBody`'s face/triangle EMISSION ORDER is nondeterministic
  run-to-run (Go map-seed randomization, e.g. `facesToFix map[int]bool`), so a positional hash FLICKERS
  even with zero geometry change (CN1 finding, confirmed pre-existing on base — a latent mesh-output-order
  nondeterminism worth its own follow-up). The order-independent sum still catches any real geometry
  change (a moved vertex → different incident triangles → different sum); pair it with per-body VOLUME
  equality to cover the winding-flip blind spot. ONLY the newly-greened cone cases may differ. Plus B3 vol
  190756.470897507 / N7 vol 963883.383205631 unchanged.
- **EXACT canal surface — no BSpline approximation.** The ruling-edge arm is a canal over a hyperbola
  spine; build it EXACTLY via the existing geom canal stack (`crossSectionArc` exact rational-quadratic
  sections + homogeneous loft over closed-form conic stations), NOT a marched/approximated BSpline the
  way OCCT did. Exactness is a requirement, not a nicety (user directive 2026-07-19).
- **C8 area override with forensic receipts — never widen the global deps.** C8's correct tangent-ball
  corner is ~+1.07% (over the 1% gate) because OCCT's own C8 corner is a 3-stripe SAG fill sitting 0.052
  off true cone tangency (NOT a rolling ball). Build the correct geometry; add a HARD-CODED per-case C8
  exception in the test suite, annotated with the receipts: the measured Δ, that OCCT's fill is
  non-tangent (0.052 off), and that Oblikovati uses the correct rolling-ball B-rep. See memory
  `occt-oracle-not-religion`. MEASURE first (CN6) — the +1.07% is a ±0.05% estimate.
- **Do-no-harm / honest-reject:** every unsupported corner declines with EXACTLY
  `"fillet: corner face must be planar"`; arm declines carry cause + offending value (the
  `sphereArmError` pattern). Concave cone bore (s=−1) honest-rejects into a follow-up slice.
- **Tessellation correctness is the highest priority.** Every newly-greened case: watertight
  (Valid+HolesContained+IsSolid, every edge 2-incident), volume-positive, and every face meshes to its
  true area with NO FOLDS (per-face DRAWEXE-oracle gate, the FR5/D9 pattern). Apply the
  `sampleCurve3OpenTrimmed` whole-curve-sub-span lesson if any sub-edge folds.
- Model-relative tolerances only (ADR-0042: `ResolutionForPoints`/`res.Weld()·scale`); angular gates as
  length bands; no bare `1e-6`. Funcs 4–20 lines; files <500; explicit types; ≤2 indent; SPDX
  GPL-2.0-only; `math.P3`.

## Tasks

- **CN1 — Arm A: torus on the cap-plane (circle) edge.** `conePlaneEdge` + `classifyConeArm` (cap-⊥-axis
  → torus; axis-containing → canal; oblique → reject) + `coneArmSurface` (offset apex A′=A+s·(r/sinα)â,
  spine circle O′, major R_s=tanα·h′, minor r) + existence guards + `torusContactCircle(geom.Cone)` case
  (s*=h·cosα+R_s·sinα). C6 reflex (270°) reuses D9's `armContactSweep`. Unit-test the derivation §2 exact
  numbers (C2 R_s 76.1257411328, D1 35, contact circles). Closed-form, small. **Does NOT green a case
  alone** (needs corner+weld) — gate: arm surface + contact circle exact vs derivation; byte-identity.

- **CN2 — Arm B: the canal band. THE CRUX.** Exact hyperbola spine (closed-form station sampler +
  tangent, parametrized by the free in-plane coord / arc length — NOT z, or D1's snout stalls), arm loft
  via `crossSectionArc` + homogeneous v-interpolation, emitted as `geom.BSplineSurface` in `edgeFillet`;
  `armStation` support. Canal FOLD guard: assert per-station regularity of the BAND arc only
  (1−κ·r·cosψ>0 across the arc), not the full tube (D1's vertex κ would spuriously reject). Gate: sampled
  band vs exact spine ≤ weld; station rails bit-on-host; trimmed-BSpline tess (`tessellate_trim.go`) meshes
  fold-free; byte-identity.

- **CN3 — Corner centre.** `coneHostCorner` + `solveConeBlend`: cos²α-scaled quadratic (qa=cos²α(d·d)−(d·â)²,
  qb, qc), **mandatory nappe filter w·â>0 before nearerRoot**, apex-singular material probe (evaluate AWAY
  from v — C8's vertex IS the apex), certificate (|dist−r|≤weld two-sided for both planes + exact signed
  cone distance), `coneTangentPoint` (meridian foot T). Dispatch ordering + byte-identity suite. Unit-test
  the §3 table to 1e-12 (C2 (75.466,−10,10) res 2.8e-14, etc.); verify `cornerCylinderArms` does NOT claim
  the Cone∧Plane ruling edge.

- **CN4/CN5 — RE-SCOPED after the CN4 attempt (2026-07-19): the CN4/CN5 split was INFEASIBLE.** The
  canal arm face cannot close its boundary loop without its far-runout cap, so greening C2/C6/D1 needs the
  canal far-cap as a hard prerequisite of the weld; and the CN2 canal arm was lofted over the edge-ENDPOINT
  span, not the corner-trimmed span. New decomposition:
  - **CN4a (DONE, commit 6dec9883):** wire the canal `armStation` (via `armCanalSpine`) + the cone
    `onHostSurface` case. Byte-identity-safe groundwork; corpus 60 unchanged. The corner SOLVE +
    great-circle weld + Gauss–Bonnet closure all pass (proving `solveCurvedCorner`/`curvedSetbackRail`
    carry over verbatim).
  - **CN4b (= merged weld + far-runout):** rebuild the canal arm over `[x_f,far, x_f,C]` (v=C boundary =
    the corner great circle); the canal FAR-RUNOUT cap (`intersectArmCapping`/`armSprings` canal case —
    oblique `canal ∩ cap-plane`; D1's hyperbola-vertex snout, endpoints bit-exact on axis + far ruling,
    guard far-plane ⊥ vertex tangent else reject); cone-host bite (torus contact-arc + canal cone-side
    rail `t↦T(m(t))` Curve3, pinch at T) + plane-host bite (exact hyperbola foot Curve3); degenerate
    pinch-at-T (do NOT synthesize OCCT's 0.14 sliver; dedup C8's two canal cone-rails). Reuse FR1–FR3 for
    the cylinder perpendicular caps + torus meridian caps. → green C2/C6/D1 (corpus 60→63). Watertight +
    volume + per-face fold gates; byte-identity.
  - **CN-C8 (separate):** C8's apex-strip / consumed-apex topology (both radial planes → bottom slivers;
    corner ball wraps over the top, excess 4.498 sr, area 448.4). Watertight + volume-positive; area
    +1.07% (red pending the CN6 override).

- **CN6 — Corpus gates + the C8 area decision.** occtparity scoreboard → corpus 64; do-no-harm across the
  60 prior greens (mesh-hash); per-case fold/watertight gates for C2/C6/C8/D1; **MEASURE C8's exact-body
  area**, and if over the 1% gate (est. +1.07%) add the hard-coded C8 test override with the forensic
  receipts (never widen global deps). DRAWEXE per-face cross-check for all four.

**Order:** CN1 → CN2 (crux) → CN3 → CN4 → CN5 → CN6. CN3 needs CN1+CN2's arms to weld; CN4 needs CN3.
Each task: fresh opus implementer + fable review + mesh-hash byte-identity + fold/watertight gates.
