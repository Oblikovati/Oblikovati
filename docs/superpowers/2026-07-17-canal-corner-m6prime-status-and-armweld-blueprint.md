<!-- SPDX-License-Identifier: GPL-2.0-only -->

# M6′ canal corner blend — status & canal-aware arm-weld blueprint

Durable capture of the M6′ milestone state at the C4 boundary. The corner-surface work (C0–C3) is
landed, reviewed, and validated on `feat/occt-blend-parity-corpus`; the whole-body weld (C4) is honestly
BLOCKED on a re-scope. This doc records **what is verified** (measured, not reasoned) and the **proposed
blueprint** for the remaining canal-aware arm-weld, so the foundation and the diagnosis survive the
milestone boundary.

## 1. Verified state (C0–C3, corpus 55, nothing wrong ships)

The rolling-ball **canal corner SURFACE** for OCCT `tests/blend/simple/N7`'s `result_5` (oracle area
**90.194**) is built and correct. All numbers below are DRAWEXE- or test-measured, each task reviewed
(spec + adversarial quality, mutation-tested).

| Task | Commit | Result (measured) |
|---|---|---|
| C0 | `845ee7a6` | `RailLoop.Canal{Rolls,Radius,Ends}` payload; `canalProvider` tier `[analyticSphere, canal, coons4, tri3]`; plate ops-stub retired; corpus 55 byte-identical. |
| C1 | `cb912e48`+`eb190f84` | `canalSpine` = inner-offset host SSI (marched; cyl∩cyl is a **skew quartic**, no analytic case). Spine C″=(55,5.279,5)→C=(45,5.279,15); on-host residual **2.37e-9** at vertices. Offset signs general (toward the ball centers). Endpoint-snap guarded. |
| C2 | `1eb4b86f`+`c8a0bb6f` | `crossSectionArc` exact rational-quadratic (radius residual **7.1e-15**). `loftCanal` homogeneous 4-D loft, u=arc / v=chord-length. **Area 90.176 = −0.020%, grid-converged, EMERGENT (no tuned constant** — the converged value ≠ the 90.194 oracle, so it is not fitted). Shape vs OCCT's 3×10 net **0.017**. |
| C3 | `69e2c56b`+`5e1b10e3` | `CanalCornerFill` + `canalProvider.Build` → `BlendKindCanal`, certificate Valid (honest, mutation-proven). Emits the canal's **own** 4 boundary isoparms (on-surface to **8.48e-13**), NOT the received rails (one received rail, `amid` at C′, was 0.28 off — a malformed-face trap, caught + fixed). B3 clean-octant → `analyticSphere` byte-identical. |

**The banked plate solver** (`kernel/geom/plate_*.go`, P0–P4a, P2.1) is correct, reviewed code for a
genuinely-variational corner but the wrong model for `result_5`; kept dormant, not deleted.

## 2. The verified corner-patch boundary topology (from C3, measured on-surface)

The canal patch (u = arc parameter, v = chord-length along the spine) has four boundaries, each a
boundary isocurve of the emitted BSpline (on-surface to 8.48e-13):

| Boundary | Isoparm | Curve | Center / host | Weld neighbour |
|---|---|---|---|---|
| v = 0 | end cross-section arc | radius-5 arc | ball center **C″** = (55,5.279,5) | adjacent arm blend (s_5 side) |
| v = 1 | end cross-section arc | radius-5 arc | ball center **C** = (45,5.279,15) | adjacent arm blend (s_4 side) |
| u = 0 | foot-locus | on host | wall cylinder **R = 50** | wall host face (retrimmed) |
| u = 1 | foot-locus | on host | s_10 cylinder **R = 5** | s_10 host face (retrimmed) |

The two END arcs already coincide with the received a0/a1 rails (measured 0.0 / 1e-14 off-surface) — those
neighbours are unaffected. The two FOOT-LOCI replace the received `[amid, E2]` u-rails; the neighbours
(wall + s_10 hosts) must be **retrimmed** to them.

## 3. The C4 blocker — non-concurrent arm spines (measured, no forced green)

The whole-body weld declines **upstream** of the wall-termination / far-runout it was scoped for, at the
**first arm face**. Root cause: `curvedWeldFaces` / `armRailBundle` (`kernel/ops/fillet_curved_weld.go`)
is founded on a **single concurrent corner ball** at `w.center = C = (45,5.279,15)` — valid for the clean
octant (B3), invalid here.

**The three arm spines do NOT concur** (the defining fact of this degenerate corner):

- s_4 offset spine: line **x = 45**
- s_10 offset spine: line **x = 55**
- s_5 offset spine: circle (radius 45 about (50,50)) at **z = 5**

`C` lies on the s_4 spine only. Concretely, the s_5 **torus** arm rolls on the wall at **z = 5** (contact
circle center (50,50,5)), while the single-ball model demands its wall-tangent at **z = 15** — a fixed

```
      Δz = 15 − 5 = 10 = 2r        (r = 5)

   single-ball center (z=15)  O
                              |\
                              | \   fixed 10-unit (2r) gap — cannot close
                              |  \
                              |   O  true s_5 torus-roll center (z=5)
                            (s_4) (s_5)
```

→ `armHostContactRail` h0 = false → *"arm rail bundle declined (geom.Torus)."* The body never assembles,
so there is no shell to weld and no Σ to compare — the weld correctly floors out rather than fabricating
a mis-closed shell. Corpus stays 55; `git diff` empty.

**Secondary finding:** `torusStation` (`fillet_curved_corner_solve.go:125`) never checks the axial offset
from the torus spine plane, so it accepts `C` at z=15 (2r off) and defers the failure to the rail bundle
instead of an honest early decline. A cheap early-decline fix.

## 4. Blueprint — the canal-aware arm-weld (the remaining ~300–500 LOC)

This is the proposed direction; the exact assembly should be nailed by a `software-architect-advisor`
consult (per CLAUDE.md) before the SDD build, because it re-shapes the arm-weld, not just a delta.

**A. Build each arm face at its OWN reflected center, not the global `C`.** For each arm boundary rail
`p(s)` with adjacent arm-surface unit normal `n(s)` and radius `r`, the native curvature center is
`Cᵢ(s) = p(s) + r·n(s)` — i.e. sweep each arm face along its own reflected-family center
(C / C′ / C″, which `tangentDegenerateSides` already computes for the fill), eliminating the single-ball
assumption that fails at the 2r gap.

**B. Weld the four canal boundaries to their four DISTINCT neighbours** (§2 table): the two end arcs to
the two adjacent arm blends (G1, at C″ / C); the two foot-loci to the wall and s_10 host faces (G1, each
retrimmed to the foot-locus). This is the topological change from the single-ball model, where all three
arms met one sphere.

**C. Reuse the far-runout machinery VERBATIM — but only after A/B exist.** `spliceCornerBite` /
`farRunoutFace` / `farArcsBiting` (the B3 far-runout path) then trim the arm faces axially at their true,
non-coincident intersections and splice the 3 notch faces (result_2/4/10). This closes the boundary loop
to the watertight whole-body Σ = **61222.9**.

**D. Apply the `torusStation` early-decline fix** (§3 secondary) so a genuinely-invalid corner declines
honestly rather than deferring.

**Order:** architect consult → arm faces at own centers (A) → 4-boundary weld + host retrims (B) →
far-runout splice (C) → whole-body watertight + Σ gate → corpus 55→56. Honest-report mandate throughout:
any un-closable seam / wrong volume → BLOCKED with the exact gap, never a loosened gate.

## 5. Options at this boundary

- **Build the canal-aware arm-weld** — finish greening N7 via §4 (architect consult → SDD). The hard,
  novel geometry (the corner surface) is done; this is the remaining assembly.
- **Bank here** — C0–C3 are durable and corpus-neutral; N7's corner resolves to the correct canal surface
  while the body weld honest-declines → coons4 (corpus 55). Revisit §4 in a fresh session.

Either way the foundation is protected: the corner surface reproduces OCCT to 0.017 shape / 0.020% area
with no tuned constant, and this doc records the exact remaining work.

## References
- Spec: `docs/superpowers/specs/2026-07-17-canal-corner-blend-design.md`
- Plan: `docs/superpowers/plans/2026-07-17-canal-corner-blend.md`
- Math (spine/arc/loft, offsets, SSI, pitfalls): `.superpowers/sdd/canal-corner-math.md` (scratch)
- Seam (payload, tier, weld gate): `.superpowers/sdd/canal-corner-seam-architecture.md` (scratch)
- Spike (−0.025% reconstruction, host-offset spine): `.superpowers/sdd/blend-sweep-spike-report.md` (scratch)
- C4 blocker detail: `.superpowers/sdd/canal-c4-report.md` (scratch)
- OCCT oracle net: `.superpowers/sdd/result5-poles.txt` (scratch)
- OCCT reference: ChFi3d rolling-ball blend; Patrikalakis & Maekawa (canal/offsets/SSI); Piegl & Tiller §10.3.
