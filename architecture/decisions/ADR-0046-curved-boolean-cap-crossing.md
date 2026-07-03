# ADR-0046 — Curved-boolean cap-crossing arrangement

**Status:** Accepted (2026-07-03). · **Advances** the EPIC
[Oblikovati#1724](https://github.com/Oblikovati/Oblikovati/issues/1724) (widen the curved-boolean
recognizer gates), the follow-up split from audit A5 ([Oblikovati#1601]). · **Builds on**
[ADR-0045](ADR-0045-curved-boolean-kind-taxonomy.md) (the T/P/D KIND taxonomy),
[ADR-0027](ADR-0027-curved-face-boolean.md) §M2 (the exact analytic curved boolean),
[ADR-0043](ADR-0043-generalized-provenance-naming.md) (curved-stitch edge provenance) and
[ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md) (model-relative tolerances). ·
**Touches:** `kernel/brep/curved_general_boolean.go` (the `loopsClearOfCaps` decline gate),
`kernel/brep/curved_general_ruled_cutjoin.go` (the cut/join drivers), a new
`kernel/brep/curved_cap_crossing*.go`, and the OCCT oracle harness
(`kernel/ops/testdata/occ_boolean_oracle.json`, `experiments/occ-boolean-oracle`).

## Context

The general ruled∩ruled curved boolean (ADR-0045, KIND **[T]** transversal) traces the wall∩wall
SSI imprint, trims each operand's periodic side band by it, classifies cells by 3-D solid
membership, and welds the kept faces plus **whole** planar caps. It **declines to a recorded CSG
facet fallback** (`CodeBooleanCSGFallback`, #1407) whenever the tool pokes **through a planar cap**
of the target — the gate `loopsClearOfCaps` (`curved_general_boolean.go`): if any imprint vertex
leaves the axial band `[vMin, vMax]`, bail.

That decline is exactly the "**partial curved-on-planar contact** (a hole clipping a face edge)"
ADR-0045 §Consequences flagged as a *separable new feature*. It is common real geometry — a
cross-drilled hole that **breaks out through an end face** — so it deserves an exact analytic
result rather than faceted soup.

The audit A5 analysis (geometry-math-advisor, recorded on #1601/#1724) established the hard
boundary: cap-crossing is **the general B-rep boolean the recognizer exists to avoid**. When the
tool pierces a cap, the wall∩wall SSI is no longer a closed loop on the band — it becomes an **open
arc on the wall** (the part with `v ≤ vMax`) that must be **closed by a conic arc lying on the cap
plane** (`tool_surface ∩ cap_plane`), and the cap disc itself must be trimmed by that conic. The
open wall arc and the cap conic arc share exactly the two vertices `V = T_wall ∩ L_wall ∩ cap_plane`
(points on the target rim circle that also lie on the tool wall). The general form couples two
conic arrangements at shared vertices with consistent manifold orientation — precisely the object
that, **half-implemented, ships manifold-but-geometrically-wrong solids** (right Euler / right
volume, wrong shape), which the internal `validBooleanSolid` + volume-bracket nets provably cannot
certify against.

## Decision

**Cap-crossing is a fourth sub-case of KIND [T], built as ONE narrow, fully-certifiable slice at a
time. The first slice is the only sub-family with a deterministic single-arc boundary; every other
cap-crossing configuration keeps the observable `CodeBooleanCSGFallback` decline. Correctness is
certified against OCCT `BOPAlgo_BOP` on an independent moment set, not only the internal nets.**

### Slice 1 — the interior-exit oblique-cylinder cap-cross

The design analysis first proposed the 2-transversal-vertex (rim-crossing) case as slice 1.
**Measuring the real traced imprint** (throwaway experiments over the candidate fixtures) revised
that: the rim-crossing case is a coupled **corner** — a separate wall entry hole PLUS a top-wall
notch PLUS a cap bite, meeting at two shared rim vertices (three interactions). The
**interior-exit** case — where the exit ellipse lies **strictly inside** the cap rim — decomposes
into two *independent* clean features and is unambiguously the simplest, lowest-risk, fully
certifiable first slice. (The mathematician's own note agrees it is "decomposable and EASIER.") So
slice 1 is the interior-exit case; the rim-crossing corner is the next slice.

Classify and build **only** the configuration where:

1. the tool is a **cylinder**, obliquely crossing the target and exiting **exactly one** planar
   cap of the target;
2. the section `tool ∩ cap_plane` is a single **ellipse** (`ε ≤ |d·n| ≤ 1−ε`, `d` tool axis, `n`
   cap normal — a circle needs coaxial, which cannot cross; a line-pair needs `d ⟂ n`, deferred);
3. that ellipse lies **strictly inside** the cap's rim circle (zero rim crossings, by a
   model-relative margin), so it is a clean **inner loop** on the cap — the oblique generalization
   of the drill-through `[P]` circle inner loop; **and** the wall∩wall imprint is **exactly one
   closed in-band loop** (the single entry hole).

Everything else — the rim-crossing corner (2-vertex), line-pair (strip / 4-vertex), re-entrant
(4-vertex), tangency, cone sections (parabola/hyperbola), a tool exiting two caps, and partial-rim
— is **declined observably**. The classifier is a positive gate: it fires only on (1)∧(2)∧(3); the
default remains the CSG fallback.

### The arrangement

- **Exit ellipse** `E = tool_cylinder ∩ cap_plane` is computed in closed form (the one curve the
  wall∩wall pipeline never produces): centre = axis∩plane, semi-minor `r` along `n×d`, semi-major
  `r/|d·n|` along the in-plane axis projection. This is **exact**; only its tessellation is
  deflection-bounded.
- **Interior test** (the slice-1 gate) samples `E` against the rim circle and requires every point
  strictly inside by `ResolutionForSize(R)`; any crossing → decline (that is the deferred 2-vertex
  corner).
- **Wall trim** reuses the existing `(u,v)` arrangement unchanged: the single in-band entry loop
  keeps the target wall outside the tool (a wall with a hole).
- **Cap trim** is trivial and exact — `E` strictly inside the rim means the kept cap is the disc
  with `E` as an **inner-loop hole** (outer = rim circle, inner = ellipse), exactly the drill
  through-hole pattern with an ellipse instead of a circle. No arrangement engine.
- **Tool tunnel** feeds `[entry-loop, E]` into the tool-wall arrangement, keeps the band **inside**
  the target (a tube from the entry hole up to `E`), and reverses it into the cavity.
- **Weld orientation contract:** every result face's boundary loop walks CCW seen from **outside**
  the solid, so every edge is shared by exactly two half-edges walked in **opposite** directions —
  the pairing `curvedStitch` keys on. The entry loop is shared by the trimmed wall and the reversed
  tunnel; the ellipse `E` is shared by the holed cap and the reversed tunnel.
- **Feed `E` by VALUE, not by pointer** (implementation invariant). `curvedStitch.edgeCurveFor`
  switches on the *concrete* curve type (`case geom.EllipseFull`) to store a re-anchored
  `EllipticalArc` whose `PointAt(0)` is the seam vertex. A `*geom.EllipseFull` misses that switch and
  the raw full ellipse is stored (its `PointAt(0)` at the major-axis vertex, not the seam vertex);
  then the holed cap (`discretizeEdge`, snaps endpoints to the seam vertex) and the tunnel rim
  (`TessellateEdge`, walks the raw domain) sample `E` at different breakpoints, T-junction-cracking
  the two faces. The crack is invisible to the internal validator (which welds the *B-rep* edge) but
  makes the *mesh* non-watertight, so `meshGeometryProperties` mis-orients across the slit and reports
  a wrong volume (238.7 vs the watertight 265.6). This is the load-bearing subtlety of the whole slice.

### The residual vs OCCT is the universal SSI-imprint deficit, not a shape error

The watertight tessellated volume lands ~0.4% under OCCT's exact analytic value (265.6 vs 266.67).
That deficit is **not** specific to this slice: the wall∩wall entry curve is a genuine
surface–surface intersection with no closed form, approximated by a fixed-resolution `geom.Polyline`
that tessellation `Quality` does not refine. Every curved boolean here carries it — measured
identical to the shipped plain crossing-cut (0.402% vs 0.385%). The *analytic construction* is exact:
a dense membership grid matches OCCT to 0.01%. So certification splits the two — the membership audit
pins the analytic shape, and the tessellated moment tolerances (~1% volume, ~0.5% area) absorb the
imprint deficit.

### Genus is configuration-dependent — never assume χ = 2

A cap-crossing **cut** (`target − tool`) opens a curved tunnel with two mouths (a wall breach + a
cap breach) → **genus 1, χ = 0**. The **intersect** (the rod plug) is genus 0, χ = 2; the **join**
is genus 0, χ = 2. The internal validator accepts any closed orientable 2-manifold regardless of
genus, so χ alone does not catch a wrong-genus build **unless the expected χ is known**. Therefore
the per-config χ is taken **from OCCT**, and the regression fixtures assert it.

### Certification protocol (mandatory for every newly-classifying config)

Validate against OCCT `BOPAlgo_BOP` + `BRepGProp` on an **independent moment set** that jointly
pins the shape, because volume + χ + face-count can all coincide on a right-volume-wrong-shape
result:

- **volume** (0-th moment), **centroid** (1-st moment — catches a wrong-side bite that volume
  misses), **surface area**, and **χ** vs OCCT;
- a **point-membership audit**: sample N points, assert
  `inside_result(p) == boolean(inside_T(p), inside_L(p))` via the closed-form oracles — a direct
  shape check independent of all moments.

The oracle is the existing precomputed harness (`occ_boolean_oracle.json`, regenerated by the C++
`experiments/occ-boolean-oracle` driver against the local OCCT checkout), extended to carry
centroid + area + χ, not only volume.

## Consequences

- **Buys:** an exact analytic result for the common cross-drilled-hole-breaks-out-the-end geometry;
  a reusable cap-crossing arrangement seam; and a certification protocol (moments + membership vs
  OCCT) strong enough to catch the "manifold-but-wrong" failure the internal nets cannot.
- **Costs:** only one narrow sub-family classifies; line-pair/strip, interior-exit, re-entrant,
  cone, tangency and partial-rim stay on the CSG fallback. Each is a further slice under #1724,
  added the same way (classify → build → OCCT-certify), never as a blanket gate relaxation.
- **Risk is bounded by the positive gate:** the classifier fires only on the exact slice-1 shape;
  any deviation falls through to the pre-existing recorded decline, so no currently-passing boolean
  changes behavior. The keep-closed-form-membership decision (do **not** adopt winding-number
  membership — imprint vertices sit *on* `∂T∩∂L`, where a ray/winding test is sign-unstable) is
  retained from #1724.

## Rejected alternatives

- **Widen `loopsClearOfCaps` to a scale-retuned margin.** Rejected: the audit proved the
  `O(band-length)` margin does not actually false-decline realistic clean crossings at the real
  coefficients (`Plane() = 1e-7·size`; only bites at aspect ratio > ~1e4:1). There is no free win
  there, and relaxing the gate without building the cap arrangement ships wrong solids.
- **Build the full conic arrangement on the cap (Bentley–Ottmann sweep of conic edges) now.**
  Rejected for slice 1: exact conic-arc arrangement predicates are a large robustness surface; the
  two-transversal-vertex case has a deterministic O(1) two-arc walk that needs none of it. The
  general arrangement is where the remaining sub-cases will live, added deliberately.
- **Switch cap membership to winding-number (A5 AC#4).** Rejected: the seam vertices are on the
  boundary `∂T∩∂L`; a winding/ray test there is sign-unstable. Keep the closed-form band+radius
  predicate.
- **Assume χ = 2 and rely on the internal manifold net.** Rejected: the cut is genus 1; asserting
  χ = 2 is the single most likely silent bug. χ is taken from OCCT per config.
