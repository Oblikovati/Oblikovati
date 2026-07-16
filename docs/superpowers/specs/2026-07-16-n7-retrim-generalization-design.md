<!-- SPDX-License-Identifier: GPL-2.0-only -->

> **SUPERSEDED (2026-07-16)** by `2026-07-16-n7-tangent-degenerate-corner-fill-design.md`. Evidence (`.superpowers/sdd/n7-c2-diagnosis.md` + `n7-runout-rederivation.md`) falsified this framing: N7 is a tangent-degenerate trihedral corner with a 4-sided rational FILL + a mis-rooted ball, not a boolean-cut retrim generalization. Kept for provenance.

# Slice B — Curved-arm fillet host-retrim generalization (N7) design

## Context

M5 Slice A shipped the all-analytic curved-arm trihedral fillet weld and greened OCCT
`tests/blend/simple/B3` — a clean 90° cylinder **wedge**. Its host retrim (`retrimCurvedHost`,
`kernel/ops/fillet_curved_retrim.go` + `_retrim_loop.go`) re-clips each host face to a single loop by
replacing the corner bite with the arm contact rails plus a "far path" around the rest of the boundary.
That retrim baked two assumptions that hold on B3's clean wedge but **break on any boolean-cut host**:

1. **The bite is on the OUTER loop.** `retrimCurvedHost` selects `Loops()[IsOuter]` and carries inner
   loops verbatim (the T5.3 review fix). FALSE on N7: the wall face has **2 wires** and the corner bite
   lands on the **inner notch-window loop**, not the outer rim.
2. **The arm ruling runs to the host's global axial extreme.** `axialExtremeEnd` slides the cylinder-arm
   ruling to the wall loop's extreme axial coordinate. FALSE on N7: the arm's true runout is the
   **filleted edge's far vertex** (z=80, against the notch-top plane), while the wall face spans z∈[0,130].
   On B3 these coincided (the wedge arm ran exactly to the rim), which is why the T5.4 cleanup safely
   unified both retrim sides on the loop-extreme version.

`solveCurvedCorner` **already succeeds on N7** (all stations + the Gauss–Bonnet closure pass — the corner
machinery generalizes). Only the retrim declines. This slice generalizes the retrim; it does not touch
the corner solve, the weld rails, or the do-no-harm floor.

The geometry-math-advisor derived the generalization
(`.superpowers/sdd/n7-retrim-generalization-derivation.md`, DRAWEXE-grounded on the vendored N7 fixture),
and every rule **provably reduces to the current B3 code when the trim is clean** — so B3 stays
byte-identical.

## Goal

Green OCCT `tests/blend/simple/N7` (whole-body area **61222.9**, `deps=1%`) — a convex R50
`[Cylinder,Plane,Plane]` trihedral corner on a `cylinder − box` cut — topology-faithful and watertight,
by generalizing the curved-arm host retrim to a **trimmed / boolean-cut host face**. Keep B3 byte-identical
and the M1–M4 corpus (S1/S4/S7/T1/T4/T7) unchanged. Corpus 55→**56**.

## N7 ground truth (DRAWEXE oracle, fixture vendored `CFI_f1234fim.rle`, `tscale ×10`, r=5)

Corner **V=(50,0,10)**; 12 faces, Σ=61222.9. The retrim-relevant faces:

| face | surface | area | role |
|------|---------|------|------|
| result_1 | Cylinder z∈[0,130] | 38033.8 | **WALL host — 2 wires** (outer rim + **inner notch loop = the bitten loop**) |
| result_6 | Cylinder | **546.695** | s_4 vertical cyl arm, z∈[15,80] (setback 15, **runout z=80**) |
| result_3 | Torus | **212.306** | s_5 torus arm, z∈[1.46,10] |
| result_12 | Cylinder | **195.464** | s_10 planar-cyl arm, z∈[10,15] |
| result_5 | (BSpline in OCCT) | **90.194** | corner sphere patch (spherical-tri excess E=3.608) |
| result_9 | Plane z=10 | **517.428** | corner host (arms s_5+s_10), 1 wire |
| result_11 | Plane x=50 | **1606.89** | corner host (arms s_4+s_10), 1 wire |
| result_2/4/10 | Plane | 1406.8 / 810.7 / 2094.6 | notch walls (runout/pass-through, retrim-untouched) |
| result_7/8 | Plane | 7853.98 ea | end caps πR² (untouched) |

## The generalization (5 changes; advisor rules R.0–R.3 + the plane-exit robustness fix)

All in `kernel/ops/fillet_curved_retrim.go` / `_retrim_loop.go`. Each reduces to current behaviour on B3.

### C0 — bitten-loop selection (R.0; fixes assumption #1)
Replace "outer loop" with: the **bitten loop `L*` = the wire of the host whose nearest vertex to the
corner-sphere centre C is minimal**, AND which carries **both** arm landings (within `res.Weld()·r`).
Retrim `L*`; carry **every other loop unchanged** (incl. the outer rim on the wall). On a plane host with
one loop, `L*` is that loop (B3 unchanged). Guard: exactly one such `L*`, else honest-reject. Extends
`bittenVertex` (`_retrim_loop.go:96`) to range over loops.

### C1 — chart-based termination (R.1; fixes assumption #2, replaces `axialExtremeEnd`)
An arm's contact curve `Γ_i` terminates at its **first forward crossing of `∂L*`**, computed in the host's
intrinsic chart so it is 1-D:
- **Cylinder wall** — chart `(θ,z)`, `θ=atan2(y−y₀,x−x₀)`. A cylinder arm's ruling is `θ=θ₀` constant
  (vertical chart segment); a torus arm's wall contact is `z=C_z` constant (horizontal). Termination =
  the nearest boundary edge of `L*` crossed beyond the corner, honouring each edge's θ-span (circle-arc
  rims map to horizontal lines; vertical notch edges to vertical lines), with θ 2π-seam-safe. **N7 s_4 → z=80.**
- **Plane** — isometric chart `(u,v)` (`planeChart`, `_retrim_loop.go:282`); ruling is a chart ray from
  tHost; termination = nearest forward `rayEdgeHit2d` (`planeRayLoopExit`).
- **Authority cross-check (R.1a):** assert `‖Γ_i(σ_far) − project_H(edge far-vertex)‖ ≤ res.Weld()·r`.
  Mismatch ⇒ first-crossing found the wrong edge, or the edge ends at an interior weld (out of scope) ⇒
  honest-reject. First-crossing is the *computation*; the filleted-edge far vertex is the *authority*.

Reduction to B3: on a clean wall with a single monotone crossing above tHost, first-forward-crossing = the
global axial extreme, so C1 ≡ `axialExtremeEnd` exactly (proof in the derivation §R.1).

### C2 — plane-exit robustness (R.1/pitfalls; the concrete `planeRayLoopExit` decline)
N7's s_4 ruling exits **exactly at a loop vertex** (z=80 junction), which `raySegment2d`'s strict
`u∈(0,1)` / `t>tol` and `segParam`'s endpoint exclusion reject → "no valid exit". Fix: test edge
**endpoints** as candidate exits, and if a landing is within `res.Weld()·r` of an existing vertex, **snap**
to it (no split — a split there makes a zero-length sliver). Grazing/collinear ruling (|d×e| below an
angular floor) falls through to C1's edge-vertex authority, not a silent drop.

### C3 — area-primary far-path on a boolean-cut loop (R.2)
On `L*` (now many edges): `insertSplits` the two landings (snap-to-vertex per C2), splitting `L*` into
`P⁺`/`P⁻`. Choose the far path by **larger enclosed chart area** (mirrors `spliceCornerBite`/`cornerBiteArea`,
`_farrunout.go:129`), oriented `L_B→L_A`. Cross-check: the discarded sub-path contains the bitten vertex v,
the kept one does not; disagreement ⇒ reject. Assemble via the existing `retrimCornerHost` (railA ∘
reverse(railB) ∘ far).

### C4 — fail-loud host-side closure invariant (R.3; the host analogue of Slice A's Gauss–Bonnet)
A signed-area balance in the host chart, gating the retrim before it emits a face:
1. **Closure/simplicity** — endpoints coincide within `res.Weld()·r`; assembled cycle is a single
   non-self-intersecting loop.
2. **Area balance** — `A_kept + A_bite = A(L*)` (chart-signed); residual > `res.Weld()·r·diam(L*)` ⇒ reject.
3. **Orientation** — `sign(A_kept)` matches the host's outward orientation (CCW outer / CW hole in chart);
   flipped ⇒ reject (would invert the face normal — the tessellation-corruption guard).
4. **Bitten-vertex partition** — v ∈ discarded, v ∉ kept.
Any failure ⇒ the clean unwelded state (do-no-harm floor), never a mis-closed loop.

## Architecture

Purely an extension of the existing retrim behind the unchanged seam: `assembleCurvedArmBody` →
`retrimCurvedHost` → (C0 loop select) → `retrimCornerHost` → (C1/C2 termination, C3 far-path) → (C4 closure
gate). The weld assembly, corner solve, rails, `filletResolvedEdges` routing, and the do-no-harm floor
(`curvedArmUnweldedError`) are untouched. No new public surface (this is GPL-module-internal `kernel/ops`).
Tolerances model-relative, corner-local `res.Weld()·r` (r=fillet radius, not body diameter); angular tests
scale-free; no bare 1e-6 (ADR-0042).

## Verification & oracle gates

- **N7 whole-body** 61222.9 (`deps=1%`) via `TestOCCTBlendSimple/N7` → PASS (corpus 55→56).
- **Per-type faithfulness** (the M3/S7/B3 lesson — topology, not just total area), fixture frame:
  the two cylinder arms `546.695` (z∈[15,80]) & `195.464`, one torus arm `212.306`, one sphere corner
  `90.194`, the two retrimmed corner-host planes `517.428` & `1606.89`, the wall `38033.8` with its inner
  loop bitten — via `countSurfaceFacesNear[T]` + the wall-loop check.
- **B3 byte-identical** — the corpus B3 subtest and B3's per-face weld faithfulness unchanged (C0–C4
  reduce to current code on the clean wedge; verify by base-vs-head diff = only N7 flips).
- **Manifold + VOLUME regression** on N7 (the orientation guard — area is orientation-blind; a flipped
  retrim face inverts the normal and fails volume even at matching area).
- **M1–M4 tripwire** — S1/S4/S7/T1/T4/T7 byte-identical (this touches only the curved-arm retrim path).
- **Corpus non-regression** — every other grid byte-identical; the convex boolean-cut family either greens
  or rides the do-no-harm floor with a clean decline.
- **DRAWEXE oracle** — the vendored N7 recipe for any per-face value; env `drawenv.sh`.

## Risks & out-of-scope (from the derivation)

- **Interior-weld runout — OUT OF SCOPE.** If a filleted edge's far end is *another* blended corner
  (σ_far interior, not on ∂H), C1's first-crossing overshoots and the far end is a weld rail to a
  neighbouring fillet (a multi-corner handoff). N7's s_4/s_5/s_10 all run out onto **unfilleted** faces, so
  N7 is safe. The plan must flag any corpus member with two blended edges sharing a far vertex → clean decline.
- **Chart seam wrap (cylinder θ).** θ is 2π-periodic; rulings/arcs near the ±π seam compare θ mod 2π
  (`wrapToSweep` semantics) — a naive compare picks the wrong crossing.
- **Inner-loop tie.** Two loops equidistant to C (pathological symmetric part) ⇒ C0 ambiguous ⇒ reject.
- **Grazing/degenerate crossing** ⇒ fall through to the edge-vertex authority, never a silent drop.
- **Concave/non-tangent hosts** untouched — this generalizes the *trim*, not the corner-solve family; a
  host the corner solve rejects still rejects.
- **BSpline corner parity.** OCCT emits N7's corner as a BSpline; our analytic sphere-triangle reproduces
  its area (90.194, E=3.608, well under the hemisphere bound). The retrim doesn't touch it; gate face 90.194
  to catch any rail mis-placement carried from the corner solve.

## References
- `.superpowers/sdd/n7-retrim-generalization-derivation.md` — the authoritative geometry (R.0–R.3, the
  chart termination, the area-balance closure, the pitfalls) and the DRAWEXE N7 ground truth.
- `.superpowers/sdd/m5-weld-setback-retrim-derivation.md` — the Slice A B3 retrim this generalizes.
- OCCT `ChFi3d` KPart / B-rep face-trimming; the DRAWEXE `blend`/`sprops` oracle is ground truth.
