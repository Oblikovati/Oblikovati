<!-- SPDX-License-Identifier: GPL-2.0-only -->

# ADR-0051 — Generalized corner-blend engine (RailLoop currency)

**Status:** accepted (2026-07-14); the corner engine ships **inside `kernel/ops/blend`, not
`kernel/blend`**, so it inherits ADR-0050's incomplete strangler migration — read
[ADR-0050 §Strangler status](ADR-0050-occt-parity-blend-engine.md#strangler-status) for what
deletes the old system and the corpus gate that unlocks it (#2200, 2026-09-01). The tier
ladders this ADR's `CornerBlendProvider` seam introduced are themselves being retired:
[#2258](https://github.com/Oblikovati/Oblikovati/issues/2258) deletes the provider tier system,
and [#3296](https://github.com/Oblikovati/Oblikovati/issues/3296)-[#3299](https://github.com/Oblikovati/Oblikovati/issues/3299)
delete the four `trihedralCornerBody` rungs — a first-fit ladder is not a classification.
Supersedes the Phase-2 scoping of
`docs/superpowers/specs/2026-07-13-curved-corner-miter-blend-engine-design.md`
(which shipped Slices 0–1 and the mid-span obstacle, ADR-4/4a). Extends, does not
replace, ADR-0050 (OCCT-parity blend engine) and the `CornerBlendProvider` seam.

**Branch:** `feat/occt-blend-parity-corpus` (standing rule: no PR until the whole
`tests/blend` corpus is green; increments accumulate + commit per task).

## Context

The `tests/blend/simple` corpus has ~69 `FAIL(faulty)` cases whose common reject
is *"corner/miter face must be planar"* (`solveBlend` fillet.go:854;
`fillet_miter.go:47/74`). The 07-13 engine greened the **planar** trihedral
corner and the **single-host mid-span obstacle**, but deferred every case with a
**curved host** at the junction. Corpus instrumentation + the S1 DRAWEXE oracle
(memory `f6-class-step-extrusion-import-defect`, `occt-blend-corner-engine-pivot`)
established that these threads — trihedral-one-curved (~30), runout-into-curved
(~16, our S1/S4/T1/T7/T9), 4-sided miter-curved (~20), n-valence planar (~8),
degenerate 3-edge/2-face (~3) — **all reduce to one requirement**: fill a
curvilinear region bounded by exact `geom.Curve3` rails, each side G1/G0 to a
known adjacent surface, certified. Extending `bsplineObstacleProvider` per-thread
would fork one geometric problem across drifting code paths.

Decision forces:
- The hardened `assembleBody`/`orientFilletShell`/weld layer must stay agnostic
  to which provider produced a patch (it consumes only `filletFace`).
- The two currently-green paths (trihedral-planar sphere; single-host obstacle)
  must stay **byte-for-byte** on their corpus cases.
- `honest-reject` (ADR-3/#1800) must remain the floor.

## Decision

Introduce a single **RailLoop** currency and one generalized dispatch behind the
existing seam, instead of per-config sibling engines.

### ADR-A — RailLoop is the request currency

A junction of any valence is expressed as one ordered, closed boundary loop:

```go
type Continuity int              // 0 = G0 (crease), 1 = G1 (tangent)
const (G0 Continuity = 0; G1 Continuity = 1)

type Side struct {
    Curve    geom.Curve3   // an EXACT boundary rail (arm end-section, host trim, or footprint arc)
    Adjacent geom.Surface  // the surface across this rail — the ARM/host geometry itself (see note)
    Cont     Continuity    // required continuity to Adjacent along Curve
}
type RailLoop struct {
    Sides      []Side        // ordered, closed (Side[i].Curve end == Side[i+1].Curve start)
    Provenance topo.Lineage  // generating tokens (root vertex + arm edge ids) for ADR-0043 naming
}
```

**Note (load-bearing, from Port Contract 1):** for a corner patch, a `Side.Adjacent`
is the fillet **arm surface** (a cylinder/cone/torus — convert to exact rational
BSpline losslessly), **not** the host face. The two arms at each patch corner
already share the host tangent plane, so G1-to-arm ribbons are twist-*compatible*
at the corners — the property that lets a polynomial fill satisfy them.

**`Adjacent` is `geom.Surface`, not `*topo.Face`** (refined during implementation):
the dependency rule below forbids a provider from importing `topo`, and a provider
needs only `NormalAt`/`DerivativesAt` off the surface to match a ribbon or recognise
a known part — so the extractor (which holds the `topo.Face`) supplies the surface
oriented material-outward, and topo identity travels on `RailLoop.Provenance`. This
also makes every provider unit-testable against synthetic `geom.Plane`/`geom.Cylinder`
with no heavy `topo.Face` fixture.

`CornerBlendRequest` keeps its `Junction/Arms/Hosts/Setback/ObstacleFeature`
fields for backward compatibility; a `Loop *RailLoop` path is added and the
existing providers are re-expressed to consume the derived loop. Rail-count
(`len(Sides)`) is the dispatch key.

### ADR-B — extraction and fill are split responsibilities

A **rail extractor** (topology → `RailLoop`, one per topology class:
`extractTrihedral`, `extractRunout`, `extractMiter`, `extractNWay`) is separated
from a **fill provider** (`RailLoop` → certified patch). The extractors own the
geometry-recognition; the providers own the surface math. Tasks 2–3 of the runout
thread (`detectRunouts`, `solveImprint`) are preserved as `extractRunout`'s rail
source. This is why one provider serves many topologies.

### ADR-C — the tier walk dispatches on rail-count and analytic recognition

`resolveBlend(loop, scale, tiers)` walks providers in order; the first whose
`Fits(loop)` is true and whose `Build` returns a `cert.Valid(scale)` patch wins.
Order (analytic-first):

1. `analyticSphereProvider` — 3 planar-tangent rails, equal radius `r`, common
   sphere center at distance `r` inside all 3 planes (the migrated `solveBlend`).
2. `analyticTorusProvider` — all rails circular arcs of minor radius `r` whose
   planes share a common axis line `L` with centers on `L` at common `R`.
3. `coons4Provider` — any 4-sided loop → one `geom.FillSurface` (Coons +
   MatchSurface ribbon-G1), per-side `Order = Cont` (the generalized obstacle path).
4. `tri3Provider` — 3-sided loop → **degenerate-4** `FillSurface`: collapse the
   corner with the most consistent arm tangents to a pole; `c0`=opposite rail,
   `d0`/`d1`=the two pole rails, `c1`=the pole point; anti-fold on interior
   stations only (v∈[0,1−δ], rotating reference normal); order recipe = `Cont`
   per real side, pole side Order 0. Gregory is a certificate-gated fallback,
   expected never to fire (corners are twist-compatible).
5. `nFanProvider` — later tier; midpoint-subdivision into n quads sharing a
   spoke-G1 central point (or Charrot–Gregory n-patch).

No provider fits / none certifies ⇒ `honest-reject` (ADR-3 preserved).

**Implementation status (2026-07-14):** the foundation wave ships tiers 1/3/4
(`analyticSphere → coons4 → tri3`) + honest-reject. Tiers 2 (`analyticTorus`) and 5
(`nFanProvider`) are **deferred promotions** — this ADR's tier *order* already reserves
their slots, and ADR-2 lets each drop in ahead of `coons4`/at the end with zero caller
change once its recognition/fill is grounded in oracle-verified corpus geometry (the
extractor-wiring phase). `coons4`/`tri3` already fill torus- and n-sided-bounded loops as
certified approximations meanwhile, so deferral is a lost *optimization*, not lost coverage.

### Consequences

- One certificate + one `FillSurface` code path for every junction class; a
  family is promoted to exact by inserting an analytic provider earlier — zero
  caller/assembly change (ADR-0050 ADR-1/2 preserved).
- The green sphere + obstacle paths are re-expressed on RailLoop and pinned by a
  byte-for-byte golden diff; `resolveBlend` sees no new topology until an
  extractor is wired (a later, oracle-gated slice), so this ADR's foundation
  wave is **corpus-neutral**.
- Cost: one currency type + an adapter on each existing provider + a load-bearing
  tier order.

### Rejected

- **Per-config sibling providers** (a curved-corner engine, a miter engine, a
  runout engine): duplicates the certificate + FillSurface plumbing across paths
  that drift; the corpus evidence shows one geometric problem, not four.
- **Rational Gregory as the default 3-sided fill**: its denominator degenerates
  at the corners where the angular residual peaks, and the twist-incompatibility
  it solves does not arise for valid fillet corners (Port Contract 1). Reserved
  as a per-family escalation only.
- **Dedicated triangular Coons over barycentric coords**: more machinery than the
  degenerate-4 `FillSurface` reuse buys, given corner twist-compatibility.

## References

Port Contract 1 (geometry-math advisor, 2026-07-14, this session): 3-sided
degenerate-4 derivation, sphere/torus recognition predicates, pole anti-fold.
Architecture brief (software-architect-advisor, 2026-07-14): RailLoop currency,
extraction/fill split, tier walk. Coons (1967); Chiyokura & Kimura (1983, Gregory
G1); Sabin/Charrot (1984, n-sided); Piegl & Tiller (NURBS, knot insertion);
Patrikalakis & Maekawa (host normal along trims). Oracle: OCCT `ChFi3d`
(`ChFiKPart` → general). Priors: ADR-0050, ADR-0042 (model-relative tol),
ADR-0043 (lineage invariance), #1800 (honest-reject).
