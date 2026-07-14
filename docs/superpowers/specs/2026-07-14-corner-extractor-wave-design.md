<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Corner Extractor Wave — Design Specification

**Date:** 2026-07-14
**Branch:** `feat/occt-blend-parity-corpus` (standing rule: **no PR until the whole
`tests/blend` corpus is green**; increments accumulate + commit per task).
**Depends on:** ADR-0051 (RailLoop currency, extraction/fill split, tier walk).
The **foundation wave is shipped** (base `7cad7cb2` → `1bd111aa`).

This document details the design, geometry math, and integration plan for the
corner-blend **extractor** wave. It bridges the topological B-rep representation
(ADR-0051 Axis A) and the generalized `RailLoop` currency consumed by the
providers (Axis B). The extractors are the wave that actually **greens corpus
cases** — the foundation wave was deliberately corpus-neutral.

---

## 1. Shipped state (what this wave builds on, not rebuilds)

The foundation wave already merged the entire `RailLoop` engine, and it is
reviewed clean with the corpus held at 50 PASS byte-for-byte throughout. **None
of the following is in scope for this wave** — it is the substrate the extractors
feed:

- `Side{Curve geom.Curve3; Adjacent geom.Surface; Cont Continuity}` +
  `RailLoop{Sides []Side; Provenance topo.Lineage}` (`corner_rail.go`).
  `Adjacent` is `geom.Surface`, **not** `*topo.Face` — providers depend on
  geom+math only.
- `railProvider{Name; Fits; Build}` + `resolveBlend`/`blendTiers()` =
  `[analyticSphere, coons4, tri3]` + honest-reject (`corner_resolve.go`).
- `analyticSphereProvider` (rails-only recognition of an exact sphere),
  `coons4Provider` (general 4-sided ribbon-G1 Coons fill), `tri3Provider`
  (3-sided degenerate-4 with pole anti-fold). Certificates measured
  (tri3 MaxDev 2.5e-15, MaxAngleDev 1.5e-8).

**Consequence for the plan:** the *provider* migration is done. The **new** work
is the **extractors** (topology → `RailLoop`), the `solveCorner` dispatch that
routes junctions through them, the **F2** ribbon-sign reconciliation, and the
**F3** certify-helper de-dup. The milestone numbering in §7 reflects this — it
does **not** re-run foundation tasks.

---

## 2. Scoping: the Tracer Bullet, executed as a Strangler

We execute **Option 3 — the Tracer Bullet** using the **Strangler Pattern**.
Rather than gate the new engine to curved-only cases (leaving planar on the old
path), we route **every** junction through one `ExtractRailLoop` → `resolveBlend`
facade, and prove transparency on the already-green cases with a **byte-for-byte
golden diff** before extending to new geometry.

Why strangler over a curved-only guard:

- **One path, one certificate.** Eliminates the dual ribbon-sign conventions that
  are the F2 landmine — the two paths stop being two.
- **The green cases become the seam test.** If `extractTrihedral` (planar) and
  `extractObstacle` reproduce the existing output byte-for-byte, the seam is
  proven transparent before any new case rides on it.

The tracer proves **both** arities and **both** fill classes through the single
dispatcher in one wave: a 3-sided analytic case (`extractTrihedral` planar →
`analyticSphere`) and the transfinite runout case (`extractRunout` S1 → **three**
valence-4 `coons4` patches tiling the interference hexagon — see §M2; the oracle
proved S1 is a double interference, not the single 4-sided quad first assumed).

**The strangler's real risk is the trim, not the surface.** Today
`spherePatchFace` emits `cb.sphere` bounded by `cb.arcs` (`chainArcs`), and
`computeFillets`/`cornerAt` trim each arm to those arcs. A byte-for-byte golden
diff requires `extractTrihedral` to reproduce **those arcs and that trimming**,
not merely re-derive an equal sphere. This dictates the bifurcation in §3.

---

## 3. Extraction geometry & the Golden Diff

The extractor's sole responsibility is translating local face-edge adjacency into
a watertight, ordered `RailLoop` of `Side` structures. Extraction logic
**bifurcates** on whether a valid prior trim exists:

### 3A. Planar-host cases — reuse the existing trim (the Golden Diff)

For existing green paths (standard planar trihedral; single-host obstacle),
`extractTrihedral`/`extractObstacle` **reuse the existing `chainArcs` and
arm-trim logic**. They must **not** recompute setbacks analytically — a
mathematically equivalent but structurally distinct arc set would fail the
byte-for-byte diff and derail the migration. Reusing the existing boundary
generation guarantees the `RailLoop` carries exactly the boundary conditions the
current `spherePatchFace` and obstacle assembler expect.

### 3B. Curved-host & runout cases — compute the setback analytically

Where no prior valid trim exists (curved host at the junction; runout crossings),
compute the setback in closed form. With `γ` = the angle between the two adjacent
host-face normals at the shared edge and `r` = the rolling-ball radius:

```
d = r · tan( (π − γ) / 2 )
```

Per arm: locate the nominal intersection `V_nom`, offset along the arm spine by
`d` to obtain the setback parameter `u_s`, and extract the **exact rational
cross-section arc** `C(v)` at `u_s` (rational quadratic Bézier for
cylinder/cone; circular for tori) to form the `RailLoop` boundary. Corner
vertices `P_k` are **shared cyclically** between adjacent arms so the loop closes
watertight.

### Golden-Diff gate (against the corrected baseline)

The full corpus run must show **zero byte-changes** in the output files of the
green cases, proving the seam is transparent — with one caveat forced by F2
(§5): the **planar-trihedral sphere** cases are byte-for-byte against their
current output (genuinely correct today), but the **obstacle** cases are
byte-for-byte against the **sign-corrected** baseline established by the M1
obstacle-sign-correction task, *not* the pre-correction output (which carries a
latent fold). Corpus PASS itself (area within OCCT 1 %, an external oracle) is
unaffected by the sign flip. Wired behind the do-no-harm fallback (§6), a
mis-extraction can never regress the green corpus.

---

## 4. Ribbon extraction: the shared normal law

Each `Side` carries the adjacent surface pointer that feeds the G1 ribbon
generator (`MatchSurface`). The extractor supplies:

- **Arm-boundary rail:** `Adjacent` = the analytical surface of the **fillet arm
  itself** (`geom.Cylinder` / `geom.Cone` / `geom.Torus`). Per Port Contract 1,
  the two arms at each patch corner share the host tangent plane, so their
  G1-to-arm ribbons are twist-compatible at the corners — the property that lets
  a polynomial fill satisfy them.
- **Host-face boundary rail:** `Adjacent` = the host plane or curved host face.

Both surfaces are converted **losslessly to rational B-spline** so `MatchSurface`
samples their **analytical normal field** along the rail (not a chord
approximation), preventing systematic normal deviation in the certificate.

---

## 5. F2 ribbon-sign reconciliation (resolved — dual-path probe, 2026-07-14)

The F2 dual-path derivation is **done** (geometry-math advisor; full report at
`scratchpad/f2-reconciliation-report.md`). The reconciliation is not "two valid
conventions" — one is proven correct and the other is a **latent bug**.

**Locked-in convention: OUTWARD `awayRef`.** `awayRef = inwardCross·(−1)`,
`dir = orientInward(n_adj×t, awayRef)`; the ribbon extrudes **outward**. Derived
directly from `geom/match_surface.go`'s `into()`: a VMin↔VMin Order-1 match sets
`(fill ∂/∂v, into interior) = −dir`, so non-fold **requires** `dir = −c_in =
awayRef = outward`. It is anchored on the fill's own Coons interior derivative
`c_in`, so it needs **no host normal** and does **not** degenerate on a flat
dihedral. coons4/tri3 are correct by construction.

**The shipped obstacle path is mis-signed.** `corner_blend_obstacle.go`
`orientInward(WallInto, inwardCrossV(base,false))` sets `dir ≈ +c_in` — the wrong
sign (fill's first station steps *out* of the patch, must fold back one control
station). It is green today only by two accidents: `creaseAngle = min(θ, π−θ)` is
antipodal-blind (so `MaxAngleDev ≈ 0` for either sign), and at 24×24 the shallow
near-seam fold does not resolve into a confirmed reversal in `obstacleNoFold`. It
passes by **sampling luck, not correctness** — "obstacle passes ⇒ its sign is
right" is **false** and must not propagate into the `coons4` route. → This is why
§7 Milestone 1 corrects the obstacle sign as its **own gated task** before the
strangler migration.

**The host-normal `σ = sign((t×n_adj)·n_patch)` is demoted to a cross-check.** It
equals the `awayRef` sign **only** on a well-conditioned convex trihedral/runout
(where `c_in ∝ +b_ref` and the trihedron keeps one handedness). It **diverges** on
saddle/mixed-convexity corners, near-flat dihedrals (its triple product → 0, sign
becomes round-off), high-curvature adjacents sampled at one midpoint, and the
**T6 obstacle-crossing class** (convex on one span, concave on the other). σ is
used only as a conditioning-gated second opinion in the probe; the fill-anchored
`awayRef` is always primary.

### The runtime dual-path probe (the M1 gate instrument)

For each **G1 side**, at the seam midpoint (rail param `lo + 0.5(hi−lo)`; tri3
pole side excluded, v-range capped at `v1 − 0.1·(v1−v0)`):

1. **Matched fill still points inward (the primary discriminator):**
   `c_in^fill · inwardCross(base) > 0` — the matched fill's into-patch
   cross-derivative agrees with the base Coons interior cross-derivative. The
   base depends only on rail *positions* (ribbon-independent), so this dot flips
   sign with the ribbon orientation; a folded ribbon gives `< 0`. Boundary-local
   and exact.
2. **`NoFold`** holds on the matched fill (interior anti-fold sweep).
3. **σ cross-check, gated:** if `|n̂_adj × n̂_patch| ≥ sin θ_min` (θ_min = 1e-3
   rad), assert `sign((t×n_adj)·n_patch) == awayRef sign`; else abstain.

> **Superseded during implementation (2026-07-14):** an earlier draft asserted
> `n̂_fill · n̂_rib > 0.1` ("same oriented normal"). That is **tautological** —
> a VMin↔VMin Order-1 match forces `F_v(boundary) = −dir` exactly, so
> `n̂_fill = −n̂_rib` *identically for both orientations* (proven + empirically
> confirmed). The boundary normal-dot cannot see the fold; check 1 above (matched
> cross vs base cross) is the real, sign-sensitive discriminator.

**Cross-path assertion (the reconciliation itself):** run **both** the obstacle
`Build` and `coons4` on the same **T6** loop; assert per-side `n_fill` agree in
orientation **after** the obstacle sign flip. Before the flip this assertion is
expected to **FAIL** on the crossing span — that failure is the proof the masked
fold was real, and is the regression test for the sign correction.

**Tolerances (model-scaled, no bare constants):** the sign/agreement threshold is
a unit-normal dot `> 0` with margin `0.1` (dimensionless); its validity is gated
on `|S_u×S_v| ≥ scale.Weld()` with `scale = ResolutionForPoints(loop corners)`
(ADR-0042) — the probe **abstains** below the floor rather than assert a sign.
Ribbon-direction / σ conditioning floors are `|n̂×·| ≥ sin(1e-3 rad)`.

**Honest-reject floor (ADR-3):** if the per-Greville sign of `w·awayRef` along a
seam is **not constant** above the degeneracy floor (a genuine within-side
convexity flip — a true saddle), no single ribbon orientation is G1-correct;
honest-reject and defer to a finer tier rather than ship a half-folded ribbon.
`awayRef` *detects* this (it sees the per-point flip); a single-point σ silently
returns one sign — the second reason to anchor on `awayRef`.

---

## 6. Dispatch guards (`solveCorner` as the strangler facade)

The entry point acts as the strangler facade. The new unified engine is tried
first; the untouched green paths remain as fallbacks only if extraction
skips/fails. Signatures below are **illustrative** — the actual wiring adapts to
the shipped `computeCorners`/`solveCorner` shapes and calls the shipped
`resolveBlend(loop, scale)` (the tier list is internal via `blendTiers()`):

```go
func solveCorner(junction) (*CornerBlendPatch, error) {
    // 1. The strangler seam (new unified engine).
    loop, err := ExtractRailLoop(junction)
    if err == nil {
        if patch, ok := resolveBlend(loop, scale); ok {
            return patch, nil
        }
        // resolveBlend honest-rejected — fall through to the legacy guards.
    }

    // 2. Fallbacks (untouched green paths if extraction skips/fails).
    if isStandardPlanarMiter(junction) {
        return solveStandardPlanarMiter(junction)
    }
    if isStandardObstacleRebuild(junction) {
        return solveStandardObstacleRebuild(junction)
    }
    return nil, ErrNoValidCornerBlend // terminal #1800 honest-reject
}
```

**Selection predicate for extraction** (`ExtractRailLoop` returns an error to fall
through, never a wrong loop):

- the junction must be **filleted** — at least one incident edge carries a
  non-zero radius;
- the boundary must be **closed** — an open boundary yields `ErrOpenLoop`, which
  routes to the #1800 honest-reject rather than a malformed patch.

The hardened `assembleBody`/`orientFilletShell`/weld layer stays **agnostic** to
which provider produced a patch — it consumes only `filletFace`. Every extractor
path is wired behind the **do-no-harm, area-improved fallback** already used by
`assembleFilletBody`: keep the new result only if it yields a valid solid whose
area improved toward parity; else the baseline.

---

## 7. Implementation milestones (Extractor Wave)

Assumes the foundation wave (rail.go, providers, dispatcher) is shipped (§1).

### Milestone 1 — F2 sign correction, F3 de-dup & the Strangler Migration

Ordered so the obstacle sign is corrected and re-baselined **before** the
strangler migration, keeping the migration a pure byte-for-byte no-op refactor.

- **Task A — F2 dual-path probe (DONE, derivation) + implement the probe.** The
  derivation is complete (§5); implement the runtime dual-path probe as a
  reusable test instrument: per-G1-side oriented-normal check
  (`n̂_fill·n̂_rib > 0.1`) + `NoFold` + gated σ cross-check, model-scaled
  degeneracy floor. This is the gate instrument for Task B.
- **Task B — correct the obstacle ribbon sign (the latent-fold fix).** Flip
  `corner_blend_obstacle.go`'s wall + two wings from `inwardCrossV(base,false)`
  to the **outward** `awayRef` reference (`…​.Scale(-1)`), matching the coons4
  `ribbonSide`. **Gate:** corpus stays ≥ 50 PASS (area vs OCCT) **AND** the
  Task-A probe passes on **T6** with the flip actually taken (cross-path assertion
  green; the pre-flip FAIL is the regression witness). This re-establishes the
  **corrected obstacle baseline** the strangler diffs against.
- **Task C — F3 certify-helper de-dup.** Unify the triplicated certify quartet
  (`tri3NoFold`/`obstacleNoFold` via a shared column-scanner taking a v-upper-
  bound; `tri3RibLen`/`coons4RibLen` via a valence-agnostic rib-length helper).
  Gated by the obstacle suite (now sign-corrected).
- **Task D — `extractTrihedral` (planar only) + `extractObstacle`.** Reuse
  `chainArcs`/arm-trim for boundary limits (§3A). Wire `solveCorner` as the
  strangler facade (§6).
- **Gate:** full corpus run yields **zero byte-changes** — sphere cases vs their
  current output, obstacle cases vs the **Task-B corrected** baseline (§3).

### Milestone 2 — Runout extractor integration (oracle-derived 3-quad hexagon tiling)

**The original single-4-sided `extractRunout` was empirically falsified.** The
DRAWEXE oracle (report `scratchpad/tracer/s1-runout-topology.md`) proved S1 is a
**double interference**: two bosses cross one fillet (footprint circles on host
planes A and B). The interfered region is a **hexagon**; our engine has only
`coons4`/`tri3`, so — exactly as OCCT does — we tile it into **3 valence-4
`coons4` patches** (central + left + right) joined by **2 shared internal seams**
(G0/watertight for this tracer; fill-to-fill G1 across the seams is a coupled
multi-patch solve deferred to M3 — the area oracle does not measure internal-seam
tangent). The single-quad guess put two coplanar sides on the same endpoints (`Closed=false`,
a flat lune). Measured constants (radius 6, ⟂ hosts): `d = r·tan((π−γ)/2) = r`;
fillet-cut abscissa `x = ±√(R_B²−d²) = ±√48 = 6.93`. **T9 is deferred to Milestone
3** — its fill is 2 patches including a valence-6 one that needs the n-sided provider.

- **Task 7 — `solveImprint` accepts `geom.Arc3d`.** Imported footprints arrive as
  `geom.Arc3d`, never `geom.Circle`; extend `footprintConic` to reconstruct the
  circle from the arc basis (unblocks S1/S4/T1).
- **Task 8 — `detectRunoutRegions`.** Cluster coupled crossings: project each
  crossing's `[pMinus,pPlus]` onto the fillet spine, merge overlapping intervals
  into one `runoutRegion` (S1's two bosses → one hexagon, not two runouts).
- **Task 9 — `extractRunout` 3-quad tiler.** Emit the 3 measured RailLoops
  (central/left/right) with 2 free-placement **shared** internal seams; **G0
  (Adjacent nil)** on the un-blended feature-arc sides (no tangent — a feature-wall
  G1 ribbon inverts the patch) AND on the internal seams (fill-to-fill G1 deferred
  to M3); **G1** on the fillet ¼-circles (→ `ef.cyl`) and host-plane runout curves
  (→ host plane). Central is thus a pure-position all-G0 Coons patch.
- **Task 10 — Wire + oracle-gate S1** behind do-no-harm; split the fillet cylinder
  outside the region and replace the span with the 3 patches.
- **Tasks 11–12 — S4/T1** (cone/torus circular footprints, reuse tiler) then **T7**
  (`solveImprint` ellipse extension; line∩ellipse).
- **Gate:** **S1/S4/T1/T7** transition to **PASS**, area matching the OCCT
  Gauss-integrated oracle to **< 1 %**; full per-case diff shows **zero
  regression** on all other cases. Corpus ≥ **54** PASS. **T9 excluded** (M3).

### Milestone 3 — Curved miter generalization, N-way & n-sided fill (incl. T9)

- **Task — `extractMiter`** (release the curved-host restriction) + remaining
  high-valence extractors; **build the n-sided transfinite fill** (subdivision or
  Charrot–Gregory) and green **T9** (valence-6 runout patch) through it; expand
  analytical setback to `n`-sided geometries.
- **Gate:** the ~69-case corner/miter block is evaluated, transitioning valid
  cases to **watertight solids** and clearing the largest failing block on the
  scoreboard.

---

## 8. Verification & oracle gates

- **Corpus non-regression (every task):** `TestOCCTBlendSimple` stays **≥ 50
  PASS**, byte-identical on untouched cases.
- **Per-family oracle:** DRAWEXE 8.0.0 built at `occt-build/lin64/gcc/bin/DRAWEXE`;
  run via `printf 'source X.tcl\n' | DRAWEXE -b`. Each newly-greened family gated
  by area within OCCT's 1 % (`checkprops`), plus per-function `kernel/ops` unit
  tests with named fakes.
- **Live test (per CLAUDE.md):** before PR, an MCP-bridge live test stresses the
  worked code path with an MCP screenshot check.

---

## 9. References

ADR-0051 (RailLoop currency, extraction/fill split, tier walk). Port Contract 1
(geometry-math advisor: corner twist-compatibility, setback, degenerate-4).
Foundation-wave final review (F2 `awayRef` derivation from `match_surface.go`
`into()`). Coons (1967); Chiyokura & Kimura (1983); Piegl & Tiller (NURBS).
Oracle: OCCT `ChFi3d`. Priors: ADR-0050, ADR-0042 (model-relative tol), ADR-0043
(lineage), #1800 (honest-reject).
