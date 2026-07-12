# V5/V1 valence-6 runout — area re-architecture (design)

**Status:** design synthesis, grounded in a DRAWEXE oracle dump of OCCT's actual V5 result +
two disconfirmed hypotheses. Follows the G5 first slice (`2026-07-12-g5-nvalent-runout*.md`).
Scope: close the residual area gap on `simple/V5` (valence-6) and `simple/V1` runouts, which
currently build valid solids but over-count surface area. Regression gate must keep `simple/V3`
(valence-5) and `simple/X9` PASS and the trihedral corpus byte-for-byte unmoved.

## Problem (measured)

Filleting a single edge whose runout ends at a >3-valence vertex closes a valid solid, but the
surface area over-shoots OCCT: after the Step-A fix below, `simple/V5` = +1.45%, `simple/V1` =
+1.72% — over the 1% corpus gate (both tripwired `FailArea`). `simple/V3` (−0.27%) and `simple/X9`
(+0.02%) PASS. The over-count is localized: for V5 the residual +356 u² is **+351 in the far-face
planes** (98.5%) and only +9 in the cylinder.

## What is already fixed (Step A — committed `63ad9205`)

`splitOnFarEdge` selected the **wrong root** of the far-edge∩cylinder intersection. A far edge (a
line) meets the fillet cylinder at two points; the old `smallestRootIn01` returned the far/back-of-
cylinder root (~11.9 cm from the apex, outside the fillet's minor-arc band) because "smallest root
in (0,1)" flips with the far edge's stored orientation. The fix selects the crossing **nearest the
apex**. This halved V5's drift (+3.24%→+1.45%) and — verified against the oracle — lands our runout
vertices **exactly on OCCT's** (OCCT vertex (41.869,89.788,50.465) = our (41.870,89.793,50.462)).
Step A is correct independent of the re-architecture and stays.

## Ground truth — how OCCT actually closes this runout (direct DRAWEXE dump)

OCCT V5 result = 8 faces = 7 planes + 1 cylinder (cylinder area 494.87; total 24551.4). By direct
introspection of its topology:

1. **The apex vertex V is DROPPED.** The original V=(42.26,90.63,50) is absent from OCCT's 12
   result vertices (nearest is 0.75 u away). V is truncated even though it lies *outside* the tube
   (`dist(V,axis)=5.573 > r=5`). *(This corrects the geometry-math-advisor's derivation, which
   concluded "V survives"; the oracle is authoritative.)*
2. **No cap, no sphere, no setback blend, no BSpline.** The cylinder's runout end is a **chain of
   4 exact `cyl∩far-plane` ellipse arcs** (minor radius = r = 5; major 7.1–23.9), one per far
   plane, joined at the 3 interior `cyl∩far-edge` near-apex vertices (each at distance exactly r
   from the axis — the Step-A splits), closed onto the 2 axis-parallel tangent lines (A-rail,
   B-rail).
3. **Each ellipse arc is ONE shared edge, used once by the cylinder and once by the trimmed,
   still-planar far face** — that direct cylinder↔far-face sharing is what welds the shell
   (weld-twice), with no cap providing the second use.
4. The far faces are **trimmed in place** by their shared ellipse edge (they stay planar, lose a
   small bite near V); V is not on any far face's boundary.

## Why the current architecture over-counts (root cause of the residual +351)

The G5 first slice welds the runout differently from OCCT: it carries **(a) a cap** (the cylinder
face's runout end is tiled from the ordered pieces via `capEndSegs`) **and (b) a separate far-face
arc piece** (`addRunoutApex` replaces the apex vertex with an arc on each far face). Each sub-arc is
shared far-face↔cap, not far-face↔cylinder. This structure was correct enough to *close* V3 (its
cap is tiny → passes 1%), but it is not OCCT's construction, and the residual far-plane over-count
is the cap + arc-piece pair fighting OCCT's single-shared-ellipse-edge trim.

**Disconfirmed hypotheses (both killed by measurement — do not revisit):**
- *Three-tier membership ("arc fewer far faces"):* OCCT arcs *zero* far faces as separate surfaces;
  it trims them. Membership count is not the lever.
- *True-ellipse arc mid (place the arc mid on `cyl∩plane`):* geometrically exact yet **regressed**
  V5 to +2.16%, because with the cap retained, sliding the mid onto the plane pushed the sub-arc
  chain axially deeper and enlarged the cap. Partial mid tweaks cannot fix it and can worsen it.

## The re-architecture (design)

Replace the cap + far-face-arc structure with OCCT's direct-share trim, for the **k=1 single-edge
runout, planar far faces** class only:

1. **Cylinder end-loop = the ordered `cyl∩far-plane` ellipse-arc chain** (`tA → q01 → q12 → q23 →
   tB`), each arc shared directly with its far face. Reuse the near-apex split points (Step A) as
   the chain's interior vertices `q`.
2. **Trim each far face by its shared ellipse edge** — the far face's loop near V becomes
   `…incoming far edge → q_entry → [ellipse arc] → q_exit → outgoing far edge…`, dropping the apex
   vertex V from the loop. No separate arc "piece" surface; the far face stays planar.
3. **Delete the cap** (`caps`/`capEndSegs` path for the fan case). The cylinder's runout end is the
   ellipse chain itself; its second use is the far faces.
4. **Drop V** from the fan apexes' loops (it is absent in OCCT). Every other (un-filleted) edge at V
   is untouched.
5. **Gate on k = (# filleted edges incident to V), not valence N.** k=1 → this construction; k≥2 →
   the existing corner/miter/blend (`addCornerRound`) path, left untouched. The current detector
   already skips blend/miter corners; confirm the fan path only ever receives k=1 corners.

### The conic-representation decision (resolve first, it gates the rest)

The shared edge is a true **ellipse** (`cyl∩plane`); `geom` has no conic type. The first task is a
**spike**: with the cap removed and the far faces trimmed directly, does a **circular arc-fit** of
the small (~21° span) sub-arc reach <1% for V5/V1, or is a true conic/spline edge required?
- The on-plane-mid data suggests the axial position of the arc matters; but that experiment kept the
  cap, so it is not decisive for the no-cap construction.
- If arc-fit suffices → done with existing primitives. If not → add a minimal `geom` ellipse (or
  rational-quadratic NURBS) curve type for `cyl∩plane`, or a tolerance-bounded multi-segment spline.
  This is a real dependency and its own decision point.

### Validity certificate (corrected)

Close weld-twice, no cap, iff: (1) each far edge has a single material-side near-apex `cyl∩edge`
crossing in the fillet's minor-arc band; (2) the crossings are strictly θ-monotone A-flank→B-flank
about the axis (non-self-intersecting chain); (3) `r ≤ r_max` (the seat still fits). V is dropped
regardless of `dist(V,axis)` (the oracle drops it even when V is outside the tube). Honest-reject
(the existing `validateRunoutFans`/#1800 path) on any failure. Carries over the monotone-order and
single-crossing tests from the shipped certificate, now run on the near-apex crossings.

### Degeneracies (model-scaled `ε = κ·min(r, ℓ_min)`, κ≈1e-7…1e-6)
- A split lands on a far-edge vertex / two splits coincide → collapse the zero-length sub-arc, weld
  the two neighbours directly at that node.
- A far face the chain never binds (its ellipse arc is empty in-band) → it is simply untouched
  (keeps full planar extent); the shell still closes on the binding faces only.
- Generator parallel to a far-face plane (`|û·n|→0`) → the section is not a simple ellipse; branch
  or honest-reject.
- Near-tangent far edge (line grazes the cylinder, discriminant→0) → single tangency, snap.

## Scope & sequencing
- **In:** k=1 single-edge runout, PLANAR far faces (V5, V1; re-baseline V3/X9 onto the same path).
- **Deferred:** quadric far faces (`cyl∩quadric` SSI edge), multi-edge (k≥2) vertex/corner blends
  (sphere/n-sided patch), variable-radius runout.
- **Gate:** V5 AND V1 within the 1% area gate (PASS); V3 and X9 stay PASS; trihedral corpus
  byte-for-byte unmoved (stash-diff); no reopened shells (every edge used twice, `IsSolid`).

## References
- OCCT `ChFi3d_Builder`/`ChFi3d_ChBuilder`, `ChFiDS_Stripe`/`ChFiDS_SurfData` — stripe/corner model:
  end a stripe at a vertex by intersecting the fillet surface with the neighbours and trimming both
  to the intersection curve, recording it as a shared edge; a closing corner *surface* (sphere) is
  emitted only for genuine multi-stripe (k≥2) vertices. Direct oracle for "trim, don't cap."
- Rossignac–Requicha (constant-radius blends as trimmed offsets); Patrikalakis–Maekawa (SSI, section
  computation, branch selection); Mäntylä (weld-twice / Euler invariants). See the G5 design spec
  and `geometry-math-advisor` report for derivations.
- Oracle harness: DRAWEXE at `../occt-build/lin64/gcc/bin/DRAWEXE`, env
  `test-utilities/occt-blend/oracle/drawenv.sh`, case `tests/blend/simple/V5`.
