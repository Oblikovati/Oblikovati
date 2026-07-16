<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Slice B (re-scoped) — N7 tangent-degenerate trihedral corner fill via the RailLoop engine

## Status & provenance

Supersedes `2026-07-16-n7-retrim-generalization-design.md`. That spec framed N7 as "generalize the
curved-arm host retrim to a boolean-cut host." Evidence-first work falsified it:

- `.superpowers/sdd/n7-c2-diagnosis.md` — on the REAL imported N7 body, `retrimCurvedHost` is **never
  reached**; the weld declines upstream in `armRailBundle`. The reverted C2 (endpoint/snap) was proven
  byte-identical pre/post — inert.
- `.superpowers/sdd/n7-runout-rederivation.md` (geometry-math-advisor) — N7's corner is a **4-sided
  rational fill** (3 arm setback rails + 1 wall-contact arc), NOT a 3-arc sphere octant, and the corner
  ball is **mis-rooted** at a tangent-degenerate dihedron (solve picks z=5 → tri area 42 vs oracle
  90.19; correct z=15). Plane `found=false` is a *symptom* of the mis-root. Reuses B3 far-runout, NOT
  the S-family imprint.
- `.superpowers/sdd/n7-corner-seam-architecture.md` (software-architect-advisor) — the 4-sided fill
  **already exists** as `coons4Provider`/`geom.FillSurface` in the RailLoop engine (System B); the
  curved-arm weld (System A) must **delegate** its corner patch to it (ports-&-adapters), not duplicate
  a second fill. B3 stays byte-faithful via the `spherePatchFace` strangler.

**Landed and kept** (corpus-neutral, reviewed): C0 bitten-loop (`81f4b190`), C1 chart termination +
far-vertex authority + weld/retrim single-source-of-truth (`859f650f`). **Reverted**: C2 inert
(`a7b62edb`). Base for this slice: **`a7b62edb`** (corpus 55).

## Goal

Green OCCT `tests/blend/simple/N7` (whole-body **61222.9**, 12 faces) — a convex tangent-degenerate
trihedral corner on a `cylinder − box` cut — topology-faithful and watertight, by (1) rooting the
corner ball correctly at the tangent dihedron, (2) **delegating** the corner patch to the RailLoop
engine so `coons4` emits the 4-sided rational fill, (3) terminating the wall arm ruling on the bitten
(inner notch) loop, and (4) splicing the three B3 far-runouts. Keep B3 byte-faithful and the M1–M4
corpus (S1/S4/S7/T1/T4/T7) unchanged. Corpus 55→**56**.

## N7 ground truth (DRAWEXE; corner V=(50,0,10); cylinder axis (50,50), R=50; Σ=61222.9)

| face | surface (OCCT) | area | role |
|------|----------------|------|------|
| result_1 | Cylinder z∈[0,130] | 38033.8 | WALL host — **2 wires** (outer rim + inner notch loop z∈[10,80]) |
| result_6 | Cylinder | 546.695 | `s_4` vertical cyl arm; setback z=15, **runout z=80** |
| result_3 | Torus | 212.306 | `s_5` torus arm; runs out onto x=80 (result_2) |
| result_12 | Cylinder | 195.464 | `s_10` planar-cyl arm; runs out onto y=30 (result_10) |
| result_5 | **BSpline deg 2×9, 4 edges/4 verts** | **90.194** | **corner FILL** — 3 arm rails + 1 wall-contact arc; verts (55.56,0.31,5),(55,5.28,10),(44.44,0.31,15),(50,5.28,15) |
| result_9 | Plane z=10 | 517.428 | corner host (arms s_5+s_10), 1 wire, 4 edges |
| result_11 | Plane x=50 | 1606.89 | corner host (arms s_4+s_10), 1 wire, 4 edges — **tangent to wall** |
| result_2/4/10 | Plane | 1406.8 / 810.7 / 2094.6 | notch walls = **far-runout targets** (result_6→_4, _3→_2, _12→_10) |
| result_7/8 | Plane | 7853.98 ea | end caps (untouched) |

## Architecture — delegate, don't duplicate (from the seam brief)

System A (`fillet_curved_*.go`, the curved-arm feature) OWNS the corner-ball solve, host retrim,
far-runout, and assembly. System B (`corner_*.go` + `geom/coons_fill.go`, the RailLoop engine) OWNS the
fill math and honest-reject via interchangeable providers walked by `resolveBlend`. They form a
**ports-&-adapters layering**: A consumes B through a new anticorruption-layer extractor.

```
curvedWeldFaces (A)
  └─ extractCurvedCorner(w cornerWeld, arms []edgeFillet, res) (RailLoop, ok)      ← NEW ACL (System A owns)
        │  octant → 3 Sides (setback great-arcs);  N7 → 4 Sides (+wall-contact arc), each Side.Adjacent+Cont=G1
        ▼
     resolveBlend(loop, res)  → analyticSphere wins the octant, coons4 wins the 4-valence fill   (System B, UNCHANGED)
        ▼
     patchToFilletFace(patch, {})  →  filletFace  →  HARDENED ASSEMBLER (agnostic to provider/Kind)
  └─ on !ok / decline: FALL BACK to curvedSphereFace (do-no-harm)
```

**ADR-1 (delegate vs extend-in-place): DELEGATE.** The analytic-vs-fill choice lives in the
`resolveBlend` tier ORDER, never in a caller-side classifier. Rejected: growing a second Coons driver
in System A (duplication + a re-introduced caller fork ADR-0051 abolishes).

**ADR-2 (B3 byte-faithful): the `spherePatchFace` strangler.** Step 1 (this slice) routes through
`resolveBlend` for the SURFACE and for the new 4-valence (N7) loop; the 3-valence (octant) branch keeps
its legacy `chainSetbackArcs` loop (surface-swap, loop-preserved) and falls back wholesale on decline —
**B3 byte-identical by construction; N7 is a brand-new branch that cannot regress anything.** Step 2
(separate follow-up, out of scope) adds a golden proving `railLoopToFilletLoops(octant) ==
chainSetbackArcs` point-for-point, then deletes `curvedSphereFace`.

**ADR-3 (public API): none.** All seam types (`RailLoop`, `Side`, `railProvider`, `cornerWeld`) are
`kernel/ops`-internal; ADR-0018 not triggered. One GPL-module PR, no release wait.

**Dependency rule (fixed):** the ACL may read System-A types + `topo`; the engine it feeds
(`resolveBlend`) stays **topo-free** (geom+math only). The assembler branches only on `filletFace`,
never on `CornerBlendPatch.Kind`/`Certificate` (telemetry only).

## The five tasks (order: T-N7.1 → T-N7.0 → T-N7.2 → T-N7.3 → T-N7.4)

Each reduces to B3 byte-for-byte on a clean octant; corpus stays 55 until N7 fully composes (T-N7.4).

### T-N7.1 — Tangent-corner ball-root selection (System A, geometry-math)
Fix `solveCurvedCorner`/`solveArmSetback` to pick the tangent-ball root whose per-arm setback stations
`spine(t)=C` lie **in-domain and adjacent to V** (z-root 15, not the reflected 5). Guard the
tangent-dihedron degeneracy explicitly (`|n_wall × n_plane| → 0`, scale-free angular floor);
honest-reject if ambiguous. **The Gauss-Bonnet closure is not sufficient** (it passed on the wrong
self-consistent triangle) — add an **area-witness**: the corner center/tangent points regression-gated
against the oracle (tangent points at z=15; corner region consistent with area 90.194). Reduces to B3
(a clean octant has one in-domain root).

### T-N7.0 — The seam: `extractCurvedCorner` ACL + delegate `curvedWeldFaces` (System A, implementer)
New file `corner_extract_curved.go`: `extractCurvedCorner(w cornerWeld, arms []edgeFillet, res)
(RailLoop, bool)` building an ordered, closed (`res.Weld()·r`) `RailLoop` — octant: 3 setback
great-arc Sides (each `Adjacent`=arm surface, `Cont`=G1); N7: +1 wall-contact-arc Side
(`Adjacent`=wall cylinder). Provenance `topo.Lineage{}` (generated by the vertex). Wire
`curvedWeldFaces` to `extract→resolveBlend→patchToFilletFace`, **falling back to `curvedSphereFace` on
`!ok`/decline** (do-no-harm). Keep the octant's legacy loop per ADR-2 Step 1. **Gate: B3 byte-identity
golden** — B3's assembled corner face unchanged (surface from the tier, loop legacy). Octant Sides must
stay concentric/equal-r so `analyticSphereProvider.Fits` still wins. The engine stays topo-free.

### T-N7.2 — The 4-sided fill: wall-contact rail + adjacency + area cert (geometry-math)
**No new fill emitter** — `coons4Provider` already fills a valence-4 `RailLoop` via `geom.FillSurface`
(Coons + per-side G1 ribbon + certify). Task: (a) in the ACL, construct the **wall-contact arc** (the
4th rail, a curve on the wall cylinder from (55.56,0.31,5) to (44.44,0.31,15)) and supply its
`Side.Adjacent` = the wall cylinder so the G1 ribbon binds to the wall; (b) confirm `coons4`'s patch
matches OCCT's deg-2×9 rational patch within the area gate — **result_5 = 90.194 at res·r²** — and G1
to all four neighbours (sample `‖n_fill − n_neighbour‖` along each rail). If `coons4`'s certificate is
too weak to guarantee the match, tighten the certificate (tracked here, not new architecture).

### T-N7.3 — Multi-loop WALL arm-ruling termination (System A, implementer)
Route `cylinderRulingOuterOnHost` (weld side) through `segsFromLoop(bittenLoop(host, w.center, tol))` —
the same bitten loop C0 gave the retrim — so edge-10's wall ruling terminates at **z=80** (inner notch
top), matching the far-vertex authority (`runoutAgrees`), not the outer rim z=130. Reduces to B3 on
single-loop hosts (bittenLoop=outer). Once T-N7.1 roots the ball at z=15, the x=50/z=10 plane exits
self-resolve (tHost back inside the loop) — verify; add the vertex-snap guard ONLY if a residual
grazing hit survives (do not pre-emptively add a plane far-runout patch — the plane retrim is provably
4-edge single-loop).

### T-N7.4 — Far-runout on 3 notch faces + DRAWEXE gate (System A, implementer)
Confirm the existing B3 far-runout `farArcsBiting`/`spliceCornerBite`/`farRunoutFace` splice each arm's
terminal cross-section arc onto its notch face (result_6→result_4, result_3→result_2,
result_12→result_10). **Gate: whole-body Σ=61222.9 (`deps=1%`) + all 12 per-face areas** (esp.
result_5=90.194, result_6=546.695, result_11=1606.89, wall inner loop bitten) + **manifold + VOLUME
regression** (the orientation guard — area is orientation-blind) + corpus 55→**56** + M1–M4 tripwire
byte-identical + base-vs-head corpus diff = only N7 flips.

## Verification & gates

- **N7 whole-body** 61222.9 via `TestOCCTBlendSimple/N7` → PASS (corpus 55→56).
- **Per-type faithfulness** (topology, not just Σ): 4-sided fill result_5=90.194 (E=3.608, NOT an
  octant); 2 cyl arms 546.695+195.464; torus 212.306; retrimmed planes 517.428+1606.89; wall 38033.8
  with its inner loop bitten.
- **B3 byte-faithful** — B3 corpus subtest + per-face weld areas + watertight + volume unchanged
  (octant keeps its legacy loop; delegating yields the same sphere). Base-vs-head corpus diff = only N7.
- **N7 VOLUME regression** — a flipped fill/retrim inverts the normal and fails volume at matching area.
- **M1–M4 tripwire** — S1/S4/S7/T1/T4/T7 byte-identical; every other grid byte-identical.
- **Corner engine untouched** — `coons4Provider`/`analyticSphereProvider`/`resolveBlend` unchanged
  except a possible `coons4` certificate tightening (T-N7.2); their existing tests stay green.
- **DRAWEXE oracle** — the vendored N7 recipe (`drawenv.sh`, `CFI_f1234fim.rle`) is ground truth for
  every per-face + volume value; record `vprops result` before hard-coding the volume.

## Risks & out-of-scope

- **ADR-2 Step 2 (collapse the octant onto the patch loop + delete `curvedSphereFace`)** — OUT OF
  SCOPE; a separate follow-up gated by the byte-identity golden. This slice keeps the octant's legacy
  loop.
- **Tangent-dihedron reflected-root trap** (T-N7.1) — the primary numerical risk; the area-witness gate
  guards it (never trust Gauss-Bonnet alone).
- **`coons4` vs OCCT deg-2×9 parity** — our Coons+ribbon patch need not match OCCT's exact BSpline
  degree, only its AREA (90.194) and G1; certified, not closed-form.
- **Interior-weld runout** — out of scope (N7's arms all run out onto unfilleted notch faces); flag any
  corpus member with two blended edges sharing a far vertex → clean decline.
- **S-family (S1/S4/T1/T7)** — NOT merged here (coplanar-boss imprint is a different topology); N7 stays
  in the corner-weld / tangent-degenerate lineage.
- **Concave/non-tangent hosts** — untouched; the corner-solve family a case rejects still rejects.

## References
- `.superpowers/sdd/n7-corner-seam-architecture.md` — the seam (ADR-1/2/3), the ACL contract, the task
  shapes, the ubiquitous language.
- `.superpowers/sdd/n7-runout-rederivation.md` — the geometry (4-sided fill, tangent mis-root,
  far-runout reuse, per-face oracle).
- `.superpowers/sdd/n7-c2-diagnosis.md` — the falsifying trace (retrimCurvedHost never reached).
- In-repo ADR-0051 (provider tiers + honest-reject + lineage invariance), ADR-0042 (model-relative
  tolerances), ADR-0043 (provenance naming), ADR-0018 (public-API split — not triggered).
