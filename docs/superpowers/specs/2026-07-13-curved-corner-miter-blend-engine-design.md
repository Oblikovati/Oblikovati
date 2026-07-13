# Curved Corner/Miter Blend Engine — Design

**Status:** approved design (architect + geometry-math consults ratified 2026-07-13; scope corrected by
corpus instrumentation to two phases)
**Branch:** `feat/occt-blend-parity-corpus`
**Goal (Phase 1, this spec):** fill the **corner-into-round** junctions — a planar-hosted cylinder
fillet whose endpoint runs into a pre-existing curved round — greening the ~16 `curvedEndpointError`
(#1797) cases without regressing the 35 PASS / planar path. **Phase 2 (separate thread):** the ~100
arm-on-curved-host CANAL-surface cases (the "…must be planar" + `curvedAdjacentError` rejects), a
continuous-sweep engine designed on its own. Instrumentation (I2 `outer=cone`, J1 `shared=cone`) showed
the "must be planar" rejects have a curved WALL on the arm — the canal class, not a junction fill.

---

## 1. Domain & ubiquitous language

The fillet engine reconstructs a valid B-rep after a rolling-ball blend. A **junction** is where
blend arms converge and the swept fillet surfaces stop short, leaving a gap a **corner/miter patch**
must fill. Today that gap is filled only when the surrounding host faces are planar; on curved hosts
(cylinder/cone/torus/b-spline) the kernel honest-rejects.

| Term | Meaning |
|---|---|
| **Junction** | Where ≥2 blend arms converge. `n==2` ⇒ **miter** (along a shared edge); `n≥3` ⇒ **corner** (at a vertex). |
| **Arm** | One converging fillet: its spine, radius, two host faces, and its end section. |
| **Setback** | Model-relative distance each arm was trimmed back so the patch has room to fill. |
| **End section** | The arm's cross-section arc at its setback end — the seam the patch welds to. |
| **Host trim** | The curve on a host face spanning between two consecutive arm ends. |
| **Corner patch** | The surface filling the junction (analytic OR fitted). |
| **Ribbon** | The prescribed cross-boundary tangent field along a side, taken from the neighbor surface. |
| **Certificate** | A provider's proof its patch is admissible: closed loop, on-surface (G0), tangent (G1), no fold. |
| **Provider / Tier** | A pluggable patch producer; the ordered tier is analytic-known-part first, bspline-general last, honest-reject after. |

The patch fills a curvilinear polygon whose boundary **alternates** arm end-sections (G1-tangent to
the fillet arm) and host trims (G1-tangent to the host). A trihedral corner is a **6-sided** region
(3 arm-ends + 3 host-trims), each side tangent to a *different* neighbor surface.

---

## 2. Architecture — one seam, an ordered tier, a certificate floor

The analytic-vs-approximation choice does not live in the engine; it lives in the **ordering of
interchangeable providers behind one seam**. This mirrors OCCT (`ChFiKPart` → `ChFi3d` general) and
lets a family start as approximation and be promoted to exact *in place*, zero blast radius.

### Integration point (2026-07-13 corpus triage + instrumentation)

Triage first bucketed rejects by message; **instrumenting the actual face geometry then corrected the
reading**. The miter/corner "…face must be planar" rejects are NOT junction fills — the picked edge's
own WALL is curved, so the rolling ball rides a curved host (I2 `outer=cone`, J1 `shared=cone`, J3
`shared=torus`). That is the **arm-on-curved-host CANAL-surface** class, the SAME geometry as the
edge-level `curvedAdjacentError` cases, and it is a **separate engine** (Phase 2 below), out of scope
here. Final classification of the 142 simple-grid `FAIL(faulty)` rejects:

| Bucket | ~count | Reject site | This engine? |
|---|---|---|---|
| **corner runs into a pre-existing round (#1797)** — planar-cylinder arm, curved junction neighbor | **~16** | `curvedEndpointError` (`fillet.go:290`) | **YES — the engine's real target, Slice 1** |
| miter/corner "…face must be planar" — a **curved WALL** on the arm | ~72 | `fillet_miter.go`, `fillet.go:834/782` | **NO — Phase 2 canal engine** (arm rides a curved host) |
| edge borders a curved wall | ~29 | `curvedAdjacentError` | **NO — Phase 2 canal engine** (same class) |
| invalid-solid ("[]" / Euler-χ / inconsistent-orientation) | ~8 | assembleBody/Validate | bugs — Y/Q Euler-χ are the *excluded* fillet-seam class; F6 "[]" is a distinct defect to probe |
| assorted setup gaps (arc-end tangency, radius range, one-radius corner) | ~17 | various | out of scope |

**Phase 1 (this spec):** the ~16 corner-into-round junction fills — the honest n-sided-hole-fill target.
**Phase 2 (separate thread):** the ~100-case canal-surface engine (a rolling ball swept along a 3D path
over a curved B-rep host — a fundamentally different, continuous-sweep pipeline with its own architect +
geometry-math consults and solver tier). Not designed here.

The seam is wired at the **`curvedEndpointError` site**, admitting a junction fill ONLY when the
**guard** holds — the guarantee we never capture a swept-arm case:

```
admit ⇔ (1) BOTH flanking walls of the picked edge are geom.Plane   // arm is a CYLINDER, not a canal
      ∧ (2) a curved face touches the endpoint that is NOT one of the edge's own walls
            AND NOT bordering any co-pick                            // a PRIOR round (#1797), not our own
```

Clause (1) alone excludes every canal case (each has a curved wall) and is self-contained — it does not
depend on the dispatch order (the miter pre-pass + `curvedAdjacentError` already filter curved-host
cases first, but the guard does not rely on that). Clause (2) is `curvedEndpointError`'s existing
`own`/`faceBordersAnyPick` logic.

```
computeEdgeFillet (fillet.go)                        ★ assembleBody / orientFilletShell ★
  ├─ curvedFilletError / curvedAdjacentError ──▶ (canal cases — Phase 2, still reject)
  └─ curvedEndpointError site:                        UNCHANGED planar path (I4, byte-for-byte)
       if bothWallsPlanar(e) ∧ prior-round-at-endpoint ─▶ resolveCornerBlend(req, tiers)
                                                    ├─ analytic-known-part providers  (later, per family)
                                                    ├─ bspline-general provider        (this spec, universal)
                                                    └─ (none certified) ─▶ honest-reject (#1800, today's error)
                                              on success: append CornerBlendPatch as a filletFace
                                                          (parent = junction provenance, ADR-0043)
```

### The seam (owned by `kernel/ops`, stable)

```go
type CornerBlendRequest struct {
    Junction math.Point3      // corner vertex; miter: shared-edge midpoint
    Arms     []BlendArm       // 2 ⇒ miter, ≥3 ⇒ corner
    Hosts    []topo.FaceRef   // read-only host faces at the junction
    Setback  Resolution       // model-relative (ADR-0042)
}
type BlendArm struct {
    Spine, EndSection geom.Curve3   // contact path; setback-end cross-section (the weld seam)
    Radius            float64
    HostL, HostR      topo.FaceRef
}
type CornerBlendPatch struct {
    Surface geom.Surface
    Loops   []filletLoop      // SAME representation assembly already consumes
    Kind    CornerBlendKind   // provenance/telemetry ONLY — never read by assembly
}
type Certificate struct {
    Closed, OnSurface, WeldsArms, NoFold bool
    MaxDev      float64        // G0 positional max deviation (model-relative)
    MaxAngleDev float64        // G1 angular tangent deviation (radians) — MUST be below seamAngularTol
}
func (c Certificate) Valid(setback Resolution) bool

type CornerBlendProvider interface {
    Name() CornerBlendKind
    Fits(CornerBlendRequest) bool                                    // cheap claim, no heavy math
    Build(CornerBlendRequest) (CornerBlendPatch, Certificate, bool)  // false ⇒ decline, try next tier
}
func resolveCornerBlend(req CornerBlendRequest, tiers []CornerBlendProvider) (CornerBlendPatch, bool)
```

### ADRs

```
ADR-1: Corner/miter surfaces come from an ordered tier of providers behind one seam;
       analytic-vs-approximation is a per-family ORDERING, never a global commitment.
  Consequences: ship all 58 on bspline-general now; promote families to exact one at a time
                with zero caller/assembly change; assembly + orientFilletShell stay agnostic;
                honest-reject remains the floor. Cost: one indirection + a load-bearing tier order.
  Rejected: commit-to-analytic (blocks curved hosts indefinitely); commit-to-bspline (permanent
            approximation everywhere, discards the exact sphere/torus patches we already build).

ADR-2: A family is promoted approximation→exact by INSERTING an analytic provider EARLIER in the
       tier; callers and assembly never change. The tier order is the single classifier — no switch.
  Guard: two providers may both Fit; the earlier wins — Fits must be precise; tier-ordering test per family.

ADR-3: A junction with no valid certificate is honest-rejected, never approximated past tolerance.
       The certificate — not "the code ran" — is the admission proof. Preserves #1800; a G0-tight
       but G1-kinked or folded patch is REJECTED, not shipped.
```

### Lineage invariance (ADR-0043 audit — RESOLVED)

Topological naming keys on the **generating lineage tokens** (root vertex + converging arm edge ids),
`ID_face = f(V_root, {E_arm})`, and is **invariant to the surface representation**. Promoting a family
from bspline to an exact sphere/torus patch changes `Surface` but not the lineage tokens — so no
downstream reference breaks. The `Kind` field is telemetry only; the stable-id generator never reads
it. **Verify before the first analytic promotion** that no per-family output smuggles surface-type
into a name key.

---

## 3. The universal B-spline fallback — math (geometry-math consult)

### The linear-G1 lever

The boundary curve already lies on its neighbor, so along a side `S_u = c_i'(t)` is automatically in
the neighbor tangent plane, and G1 reduces to the **single scalar linear condition**

```
S_v(γ_i(t)) · ν_i(t) = 0        (ν_i = neighbor unit normal along side i)
```

Linear in the control points → a *deterministic constrained least-squares solve*, not a nonlinear
optimizer. This is why the polynomial fallback is preferred over rational Gregory (whose denominator
degenerates at exactly the corners where the angular residual peaks) — Gregory is reserved as a later
per-family analytic-ish tier, triggered by a stubborn corner-twist residual.

### Seed + certify, on VERIFIED existing tooling

Kernel assets confirmed present (do NOT reinvent):
- `geom.CoonsFill(c0,c1,d0,d1)` — bicubically-blended Coons, G0-exact over 4 boundary curves.
- `geom.FillSurface(c0,c1,d0,d1, [4]FillSide{Adjacent,AdjEdge,Order})` — Coons + `MatchSurface` to
  impose G1/G2 to each **adjacent surface** across each side. This is the ribbon-native fill.
- `geom.MatchSurface(s,t,sEdge,tEdge,order)` — cross-boundary G1/G2 row matching to a neighbor.
- `ops.FillNSided(neighbours, order)` / `FillFourSided` — n-sided opening → four logical sides → fill.

**Construction, split by valence:**
- **n=2 miter and any 4-sided junction → single `FillSurface`** with the 4 boundary curves (2 arm-ends
  + 2 host-trims) and 4 `FillSide{Adjacent: arm|host surface, Order:1}`. G1 on all four sides is
  *native* — no new solver. Unequal-radius miter is just two arm arcs of different curvature.
- **n≥3 (alternating 6-sided corner) → n-sided fill**, BUT the existing `FillNSided` documents a
  **G0-only fallback on split/merged sides** (`fill_nsided.go:19-21`) — the twist-compatibility limit.
  This is the case that needs the ribbon-constrained certify/refine below.

**Certify loop (the dense oracle):**
1. Build the seed surface (above).
2. Sample each side at **curvature-adaptive** density `m_i ∝ κ_max(neighbor along c_i)·len(c_i)`
   (flat sampling certifies false G1 on high-curvature hosts).
3. On a 2× independent dense sample measure `MaxDev`, `MaxAngleDev = max_i max_t ∠(n_patch, ν_i)`, and
   the **fold check** `min (S_u×S_v)·n̄ > 0` (no Jacobian sign reversal — the anti-fold gate, the
   numeric analogue of the orientation invariant just hardened in B2).
4. If all within tol → **certify**. Else **h-refine**: knot-insert where the angular residual peaks
   (usually a corner), re-solve. Cap 3–4 iterations. Plateau above `seamAngularTol` or unfixable fold
   → **honest-reject**.

**The KKT/fairing solve is DEFERRED.** No sparse solver exists in-kernel; `MatchSurface` already does
the per-side linear G1 matching. Build the explicit equality-constrained QP
(`[λK Aᵀ; A 0][P;μ]=[0;b]`, G1 rows in `A`, bending energy `K`) **only** for the n≥3 sides where
MatchSurface's G0 fallback fails the certificate — as a documented escalation, in-house small dense
symmetric-indefinite solve (no third-party dependency).

### Numerical guardrails (each → an explicit provider check)

| Pitfall | Guard |
|---|---|
| Corner twist incompatibility | Do NOT hard-constrain `S_uv`; let LSQ absorb it; residual peaks at the corner → h-refine or reject. **A stubborn corner residual is the telemetry signal to promote the family to a Gregory analytic tier.** |
| Fold / self-intersection | Hard, separate certificate gate on the sign of `S_u×S_v`; the B2 regression class must not recur. |
| Unequal-radius miter | Per-side ribbon scaling `s_i ∝ r_i`; no symmetry assumption. |
| Near-tangent arms / small setback (sliver) | Set `setback ∝ r`; if region min cross-dimension `< k·w` (sub-tessellation, P2 sagitta logic) → `Fits` returns false / reject before assembling the matrix. |
| High host curvature (cone apex, torus inner ring) | Curvature-adaptive station density; detect under-sampling when dense MaxAngleDev ≫ constraint-station residual → densify. |
| Valence > 4 conditioning | Curvature-weighted centroid; monitor solve condition number; reject a folded rosette rather than emit it. |
| Boundary-on-neighbor assumption | Evaluate `ν_i` on the ACTUAL neighbor at the projected point; fold b-spline-host approximation error into MaxDev. |
| Tolerances | ALL model-relative (ADR-0042): `MaxDev tol = k·w`; `MaxAngleDev tol = seamAngularTol` (scale-free radians); `setback ∝ r`; sliver floor `∝ w`. No bare constants. |

---

## 4. Implementation slicing (corpus-gated, commit per slice to the branch)

Each slice: increment on `feat/occt-blend-parity-corpus`, gated by the scoreboard (cases migrate
faulty→area/PASS, **never** regress), plus the ops suite + corpus gating green. NO PR until the whole
corpus is green (branch discipline).

- **Slice 0 — the seam (zero behavior change). DONE (commit 7946559a).** `corner_blend.go`: port,
  Request/Patch/Certificate (incl. `MaxAngleDev` + `NoFold`), `resolveCornerBlend`, `Certificate.Valid`
  + 5 unit tests. No wiring yet — request construction lands with its consumer (Slice 1). Scoreboard
  unchanged (35/18/142).
- **Slice 1 — corner-into-round via `curvedEndpointError` + certificate (~16 cases).** Wire the
  `curvedEndpointError` site behind the `bothWallsPlanar(e) ∧ prior-round-at-endpoint` guard: build a
  `CornerBlendRequest` from the planar-cylinder arm(s) + the pre-existing round neighbor, run
  `resolveCornerBlend`, drive `geom.FillSurface` (native G1), certify (MaxDev/MaxAngleDev/fold),
  certify-or-honest-reject. **Gate:** the corner-into-round subset migrates faulty→area/PASS; zero
  regression; planar corner path byte-for-byte unchanged (I4). Start with the low-valence subset; escalate
  to the ribbon-constrained certify/refine (in-house KKT) only if the alternating-side G1 needs it.
- **Slice 2 — hardening + n-valent corner-into-round.** Higher-valence junctions among the ~16; pitfall
  guards as tests; the certify/refine loop where MatchSurface's G0 fallback can't hit G1.
- **Later — analytic-known-part promotions.** Per family, as the geometry-math advisor derives exact
  surfaces. Each promotion = one provider file + one line in the tier constructor + a tier-ordering test.
  Assembly and callers untouched.

**Out of scope for this engine — Phase 2, a separate architectural thread:** the ~100 arm-on-curved-host
CANAL-surface cases (the ~72 miter/corner "must be planar" + ~29 `curvedAdjacentError`) — a rolling ball
swept along a 3D path over a curved B-rep host, a continuous-sweep pipeline that gets its own design, math
consult, and solver tier. Also out: the ~8 invalid-solid defects (Euler-χ Y/Q = excluded fillet-seam
class; F6 "[]" a distinct bug); assorted setup gaps.

---

## 5. Handoff contract

**Invariants (owner):**
- **I1 — Weld:** patch trim loop welds to every arm `EndSection` within setback (else reject). Owner: `Certificate` via `resolveCornerBlend`.
- **I2 — Agnostic assembly:** `assembleBody`/`orientFilletShell` never branch on provider or `Kind`; they read only `filletFace`. Owner: the port output shape.
- **I3 — Honest-reject floor:** no junction emitted with an invalid certificate. Owner: `resolveCornerBlend`.
- **I4 — Planar/trihedral path frozen:** only curved-host junctions (the two current curved rejects) enter the seam. Owner: the `computeEdgeFillet` dispatch guard. *This is the byte-for-byte guarantee.*

**Dependency rules:** providers depend on `geom`+`math` only; the engine depends on the provider
interface; `fillet.go` depends on the engine. Nothing new depends on `assemble_curved.go`; it depends
on nothing new. A provider must never reach into assembly or the orientation pass.

**Free vs fixed:** free — everything inside a provider (fit algorithm, tier contents, the deferred KKT
internals). Fixed — the port signature, certificate semantics, the honest-reject floor, the
`filletFace` output shape, and the planar/trihedral path (I4).

**Tests that pin the seam:** tier-ordering test per promoted family (ADR-2); certificate-rejection test
(an intentionally-bad fit is rejected, I3); fold-gate test (a folded candidate fails NoFold); the
scoreboard as the coverage gate (faulty→area/PASS, never regress).

---

## 6. References

Coons (1967) blended patches; Chiyokura & Kimura (1983) G1 Gregory vertex blends (the promotion
tier); Gordon (1971) transfinite interpolation (rejected here); Sabin/Charrot (1984) & Hahn (1989)
n-sided subpatch fills; Piegl & Tiller *The NURBS Book* (knot insertion / degree elevation);
Patrikalakis & Maekawa (host normal/curvature along trims). Oracle: OCCT `ChFi3d` (`ChFiKPart` →
general). Kernel priors: M36-F07 #1300 (`geom.FillSurface`/`FillNSided`), ADR-0042 (model-relative
tolerances), ADR-0043 (provenance naming), #1800 (honest-reject), B2 (winding/fold invariant).
