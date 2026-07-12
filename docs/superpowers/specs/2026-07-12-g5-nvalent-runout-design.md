# G5 — Single-fillet runout at an n-valent vertex (design synthesis)

**Status:** design synthesis from two advisor passes (geometry-math-advisor + software-architect-advisor),
reconciled. Feeds a `writing-plans` implementation plan. Regression gate: `simple/V3` (runout vertex
valence 5) and `simple/V5` (valence 6). Grounded in ADR-0050 and the confirmed root cause
(roadmap `2026-07-11-occt-blend-greening-roadmap.md`, "V3/V5 root cause = fillet runout at a >3-valent
vertex").

## Problem (confirmed empirically)

Filleting one edge `e` (between planar faces A, B) with a rolling ball radius `r` runs out at each end
vertex. The shipping reconstruction assumes that vertex is **trihedral**: it designates ONE far face
`E` (the first face that is not A or B — `endFaceAt`, fillet.go:679) to carry the whole `tA→tB` runout
arc, pulls A and B back to their tangent points, and leaves every other incident face an untouched
survivor. At a vertex of valence `N>3` the extra far faces keep the original vertex while A, B, and the
one `E` move — shared edges no longer weld → open shell (a silent boolean/volume landmine). The runout
end-section must instead be **distributed across the whole far-face fan**.

## The math (geometry-math-advisor)

Frame at the vertex `V=0`. Faces `F₁…F_N` are planes through `V`; `e=g₁=A∩B`. The fillet **cylinder**
`C` has radius `r` and axis `ℓ = A_r ∩ B_r` (the inward offsets of A, B by `r`), direction
`u = (n_A×n_B)/|n_A×n_B|`. The runout end-section is **`C ∩ (F₃ ∪ … ∪ F_N)`**, running `tB → tA`.

- **Split point on each far edge `g_k = τ·d_k`:** solve the quadratic
  `A_k τ² + 2B_kτ + C = 0`, with `A_k = 1−(d_k·u)²`, `B_k = d_k·b − (d_k·u)(b·u)`,
  `C = d²(V,ℓ) − r²`. The physical crossing is the root `τ_k ∈ (0, L_k)` **on the material side**
  (signed test `n_j·x ≥ 0 ∀j` — critical for reflex/concave far edges; do NOT blindly take the
  smallest positive root).
- **Piece on each far face `F_k`:** `C ∩ plane(F_k)` is a **conic** — a **circle only when `u ⟂ F_k`**
  (the axis-perpendicular planar case the shipping code hard-codes), **an ELLIPSE generically**, and a
  **space quartic when `F_k` is a quadric** (needs SSI). ⚠️ **This is the first tension — see below.**
- **Join continuity:** consecutive pieces meet **C⁰** at the shared split point (correct — the far
  edges are un-blended creases, so demanding tangency there would be wrong); `G1` only at the rails
  `tA, tB` where the cap is tangent to A, B (automatic).
- **Membership (the actual fix):** each far face receives the sub-arc between its two edge crossings,
  ordered by angle `θ` about `V`. Generically all `N−2` far faces get one piece (V3 N=5 → 3, V5 N=6 →
  4). A face is **skipped** (keeps `V`) when the tube doesn't enter its wedge. **Update EXACTLY the set
  the membership test returns — never a fixed count.** ⚠️ **Second tension — see below.**

### Validity certificate (honest-reject predicate — the n-valent #1800)

Emit a shell **iff all** hold, else reject the fillet (or reduce `r`):
1. **Seat exists:** `P_r = ⋂_k{c : n_k·c ≥ r} ≠ ∅` near `V` (⟺ `r ≤ r_max(V)`).
2. **Single physical crossing per far edge:** exactly one material-side root in `(0, L_k)`.
3. **Monotone angular order:** `θ(tB) < θ(g₂) < … < θ(g_N) < θ(tA)` (non-self-intersection).
4. **Sector + solid containment:** each sub-arc lies in its face wedge and satisfies `n_j·x ≥ 0 ∀j`.
5. **Rail validity:** `tA, tB` fall in the still-valid trimmed regions of A, B.

### Degeneracies to guard (model-scaled, `ε_len = κ·min(r, min L_k)`, `κ≈1e-6…1e-4`)
- crossing lands on a far-edge vertex / two crossings coincide → drop the degenerate piece, snap to the
  existing vertex, weld neighbours directly (never a zero-length edge);
- far face tangent to the tube (conic discriminant `|disc| < ε_len·r`) → no 2D piece, grazing contact
  only (this is the class of the bug we just fixed in G2 — tangency is real here);
- reflex far edges → signed material-side root selection;
- `r` overruns the 1-ring (`τ_k ≥ (1−ε)L_k`) → local model invalid, reject/reduce or propagate;
- near-parallel axis/edge (`A_k → 0`) → linear branch.

## The seam (software-architect-advisor)

A three-stage pipeline, hexagon around a pure solver:

```
topo ─▶ DETECTOR ─▶ endCornerFan ─▶ SOLVER ─▶ runoutSpread ─▶ REBUILD ─▶ topo
      (fillet_runout_fan.go)  (value)  (fillet_runout_spread.go)  (value)  (fillet_faces.go)
```

- **DETECTOR** `classifyEndCorners(body, fils)` — the SOLE router: fan size 1 → the existing trihedral
  `ends`/`addCornerRound` path (untouched); fan size ≥2 → the new `spreads` path. Reads topo, emits pure
  value objects.
- **SOLVER** `solveRunoutSpread(fan endCornerFan) (runoutSpread, error)` — the math above. Imports ONLY
  `geom` + `math` (never `topo`). Honest-rejects via `error`. This is the box the math fills.
- **REBUILD** — `transformLoop` gains **exactly one** new arm (`addSpreadPiece`, ≤6 lines). Far-edge
  split points ride the **existing `#695 inserts` weld channel** keyed by edge id, so "split shared by
  two faces, welded twice" is inherited from proven code and `curvedSolid` passes for free.

**Value objects (fixed contract):** `endCornerFan{FilletEdge, FaceA/B, Radius, Center, Axis, Apex,
TA, TB, Fan []fanFace, FarEdges []fanEdge, Winding}` in; `runoutSpread{Pieces map[faceID]cornerPiece,
Splits map[edgeID]Point3, Valid}` out; `cornerPiece{Curve geom.Curve3, TIn, TOut}`.

**ADRs:** (A) n-valent runout is a pre-pass producing a face→pieces map, NOT a branch in `transformLoop`
(cross-face split consistency must be computed once, centrally — computing it per-face is the exact bug).
(B) the seam is a topology-free value object; the solver never sees `*topo.*`. (C) far-edge splits reuse
the `#695 inserts` channel. Honest-reject propagates through `filletResolvedEdges` (where `#1800` errors
already flow), so no signature surgery on `filletResultFaces`.

**Keep the trihedral path byte-for-byte** — it is heavily regression-pinned; unify only the *concept* and
the *dispatch point*, not the implementation. `fillet_faces.go` is already 584 lines (over the 500 budget)
→ solver/detector go in NEW files.

## Two tensions where the briefs meet (my reconciliation)

### Tension 1 — the piece is an ELLIPSE, not a circular arc
The architecture's `cornerPiece.Curve geom.Curve3` was framed on the trihedral analogue
(`addCornerRound` emits a circular arc). The math says the per-face piece is a **circular arc only when
the fillet axis is ⟂ the far face**; **generically an ellipse** (planar far face) and a **quartic** on a
quadric far face. **V3/V5's far faces are planar but not axis-perpendicular → their pieces are elliptical
arcs, not circles.** So even the minimal V3/V5 fix needs the seam to carry a conic. Options for the plan:
(a) add a `geom` ellipse/conic curve type; (b) approximate each piece by a tolerance-bounded circular-arc
spline (matches how the tessellator already discretises curved edges) — likely the pragmatic first slice.
**Decision needed:** conic curve type vs. arc-fit approximation for the first slice.

### Tension 2 — a skipped far face adjacent to a piece face breaks the weld
The math says a far face can be **skipped** (keeps `V`) while its neighbours receive pieces. But if face
`F_k` trims its `V` to a split point on shared edge `g_k` while the adjacent skipped face `F_{k+1}` keeps
`V`, the two sides of `g_k` no longer coincide at the `V` end — **the exact open-shell bug, one edge
over.** Neither brief cross-checked this. Resolution: a skipped face **adjacent to a piece face along a
split edge is not a pure survivor** — its `V` must still be pulled to that edge's split point (a
degenerate "piece" that is just the pullback, no arc). I.e. membership has two tiers: *receives an arc*
vs *receives only a split-point pullback* vs *fully untouched* (all its edges un-split). The detector/
solver contract must distinguish these so every split edge welds twice. **This must be an explicit
invariant in the plan**, tested directly on V3 (where face 28/31 are the skipped-or-pulled neighbours).

## Scope & sequencing

- **First slice = V3/V5:** single-edge runout, **planar far faces**, elliptical-arc pieces. This
  sidesteps the quartic-on-quadric SSI (defer quadric far faces to a later slice) and exercises the full
  n-valent machinery (detector, solver, membership, split-welding, honest-reject) on the two confirmed
  cases.
- **Regression gate:** V3 and V5 turn PASS (valid closed solid, area within OCCT 1%) and STAY green;
  the trihedral corpus cases must not move (byte-for-byte path + stash-diff of the PASS set).
- **Deferred:** quadric far faces (SSI quartic pieces); multi-edge vertex blends (setback, Method 3 —
  the correct escalation when >1 edge at V is blended, out of scope here).

## Open decisions for the plan
1. Tension 1: conic curve type vs arc-fit approximation for elliptical pieces (recommend arc-fit first).
2. Tension 2: the three-tier membership (arc / split-pullback / untouched) as an explicit welding
   invariant, tested on V3.
3. Whether `classifyEndCorners` subsumes the trihedral `ends` build or only routes to it (recommend:
   only routes — keep `ends` byte-for-byte).

## References
Rossignac–Requicha 1984 (rolling-ball blends); Choi–Ju 1989; Vida–Martin–Várady 1994 (blend survey);
Maekawa 1999 (offsets/self-intersection); Patrikalakis–Maekawa 2002 (SSI for quadric far faces);
Hoffmann 1989 / Mäntylä 1988 (B-rep validity). Oracle: OCCT `ChFi3d_Builder::PerformSetInDS`/
`PerformCorner` (valence-dispatched corner processing; `OnSame`/`OnDiff` = the membership question;
`SetRegul` = per-seam continuity flags). Setback vertex-blend citation (Várady/Rockwood) unverified —
confirm before use.
