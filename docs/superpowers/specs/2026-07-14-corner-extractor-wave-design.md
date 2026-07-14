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
`analyticSphere`) and a 4-sided transfinite case (`extractRunout` S1 →
`coons4`).

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

### Golden-Diff gate

The full corpus run must show **zero byte-changes** in the output files of the
existing green planar-trihedral and obstacle-rebuild cases, proving the seam is
transparent. Wired behind the do-no-harm fallback (§6), a mis-extraction can
never regress the green corpus.

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

## 5. F2 ribbon-sign reconciliation (mandatory pre-flight)

Before any legacy obstacle-class geometry routes through the `coons4` transfinite
solver, the ribbon-sign evaluation must be unified. A real static divergence
exists between the two shipped conventions:

- **Shipped foundation convention** (coons4/tri3): ribbons extrude **outward**,
  `awayRef = inwardCross·(−1)`. The foundation's final review **proved** this
  correct by deriving it from `geom/match_surface.go`'s `into()` sign (a
  VMin↔VMin Order-1 match negates the ribbon cross-derivative, so outward is
  required for a non-folding seam).
- **Shipped obstacle convention** (`corner_blend_obstacle.go` `orientInward`):
  ribbons extrude **inward**, with identical `MatchSurface` parameters, yet is
  empirically green. It is masked because `seamCrease` uses
  `creaseAngle = min(θ, π−θ)`, which is **sign-insensitive**; only `NoFold`
  guards sign, and each path passes on its own geometry.
- **Theoretical host-normal anchor** (proposed): with `n_adj` = adjacent outward
  unit normal, `t` = unit rail tangent `C'(0.5)`, `n_patch` = host-face normal
  pointing **into** the patch interior, `b_ref = t × n_patch`:

  ```
  σ = sign( (t × n_adj) · n_patch )
  ```

  passed to `MatchSurface` so `S_v` points consistently inward relative to the
  patch boundary.

**This σ formula is NOT oracle-backed against the shipped Go foundation.** It is a
standard differential-geometry derivation, but the foundation locked in `awayRef`
natively. Assuming they are identical is exactly how the just-eradicated fold
returns. It anchors on the **host normal** `n_patch`, whereas the shipped
coons4/tri3 anchor on the **plain-Coons interior cross-derivative** — a different
anchor.

**Mandatory pre-flight guard (Milestone 1, first task):** the geometry-math
advisor executes a **dual-path derivation** proving the mapping between `awayRef`
and `σ` on the shipped `coons4`/`tri3` fixtures and the obstacle **T6** case, and
specifies the **runtime dual-path probe** (feed one loop through both ribbon
builders; assert the same oriented normal field / both `NoFold`). The wave locks
into whichever convention reproduces the non-folding `awayRef` behavior the
foundation verified. No obstacle geometry touches `coons4` until this is proven.

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

### Milestone 1 — F2/F3 reconciliation & the Strangler Migration

- **Task A — F2 dual-path probe** (geometry-math advisor). Derive the mapping
  between `awayRef` and the host-normal `σ` on the shipped `coons4`/`tri3`
  fixtures and obstacle T6; specify + implement the runtime dual-path probe;
  document the proven ribbon-sign convention the wave locks in.
- **Task B — F3 certify-helper de-dup.** Unify the triplicated certify quartet
  (`tri3NoFold`/`obstacleNoFold` via a shared column-scanner taking a v-upper-
  bound; `tri3RibLen`/`coons4RibLen` via a valence-agnostic rib-length helper).
  Enabled by convergence; gated by the obstacle suite.
- **Task C — `extractTrihedral` (planar only) + `extractObstacle`.** Reuse
  `chainArcs`/arm-trim for boundary limits (§3A) to secure the golden diff. Wire
  `solveCorner` as the strangler facade (§6).
- **Gate:** full corpus run yields **zero byte-changes** in the output files of
  the existing green planar-trihedral and obstacle-rebuild cases.

### Milestone 2 — Runout extractor integration (the Tracer Bullet)

- **Task — `extractRunout`.** Reuse the preserved `detectRunouts`/`solveImprint`
  (Tasks 2–3) as the rail source. Compute curved analytical setbacks (§3B) where
  needed; capture crossing nodes `P±`; trim fillet arms axially; generate the
  4-sided `RailLoop`.
- **Gate:** the 5 area-discrepancy cases (**S1/S4/T1/T7/T9**) transition to
  **PASS**, surface area matching the OCCT Gauss-integrated oracle to **< 1 %**;
  full per-case before/after diff shows **zero regression** on all other cases.

### Milestone 3 — Curved miter generalization & N-way

- **Task — `extractMiter`** (release the curved-host restriction) + remaining
  high-valence extractors; expand analytical setback to `n`-sided geometries.
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
