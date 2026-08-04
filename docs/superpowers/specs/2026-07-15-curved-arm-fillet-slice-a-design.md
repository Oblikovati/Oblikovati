# Curved-Arm Fillet — Slice A (axis-aligned Plane∧Cylinder host) design

<!-- SPDX-License-Identifier: GPL-2.0-only -->

## Context

The OCCT blend-parity corpus has ~67 curved-corner reds that two M5 spikes proved are **one
geometry problem, blocked upstream**: the kernel cannot fillet a **curved edge** where a plane
meets a cylinder (`curvedFilletError`, `fillet_curved.go:42`, *"why a cylinder+plane edge cannot
(yet) be rounded"*). The 32 "trihedral-1-curved-host" reds are not a corner-patch job — at each
valence-3 `[Cylinder,Plane,Plane]` vertex, two of the three picked edges are Plane∧Cylinder curved
edges whose arm fillets *are* two of the corner's three rails. The corner reject (`fillet.go:914`)
runs before the arm reject (`computeCorners` at `:152` precedes `computeFillets` at `:156`) and
**masks** it; removing the mask fails all 32 at the arm. So the dependency is inverted: the
**curved-host arm fillet must land first**, then the corner assembles from real rails. This arm
capability is shared with the 35 "curved-ARM" reds → the two buckets are the same geometry.

The geometry-math-advisor derived it (`.superpowers/sdd/m5-curved-arm-derivation.md`), grounded in
DRAWEXE ground truth on the actual B3 fixture (`sprops result = 20559.5`, the exact corpus area).
The verdict: for the **axis-aligned** cylinder host, every surface OCCT builds is an **exact
analytic primitive already in our kernel** — no canal surface, no BSpline, no new surface type, no
solver, no new tessellator. That is Slice A.

## Goal

Build the constant-radius rolling-ball arm fillet on an **axis-aligned Plane∧Cylinder edge** (the
edge is a circle or a line, not an ellipse) and the equal-radius trihedral corner it feeds, greening
the axis-aligned `[Cylinder,Plane,Plane]` family — **B3 (20559.5), N1 (58091.9), O1 (65104.9)** and
siblings — to the OCCT area oracle (`deps = 1%`), topology-faithful and watertight. The oblique /
elliptical-edge case (a genuine canal surface) is **firewalled to a later Slice B**.

## The geometry (from the advisor derivation; DRAWEXE-verified on B3)

A rolling ball of radius `r` kept tangent to a plane `P` (normal `n̂_P`) and a cylinder `C`
(axis `â`, radius `R`) has its centre on the **spine** `γ = P_r ∩ C_ρ`, where `P_r` = `P` offset
by `r` to the material side and `C_ρ` = `C` offset to radius `ρ = R∓r`. The fillet surface is the
radius-`r` pipe (canal) about `γ`. Classify the edge by `s = â·n̂_P`:

- **Config (i) — axis ⊥ plane (`|s|≈1`), edge = circle.** `γ` is a circle of radius `ρ` ⟹ the arm
  is an **exact `geom.Torus`**: `MajorRadius = R∓r`, `MinorRadius = r`, axis `â`, centre = `A`
  projected onto `P_r`. Convex (protruding rim) ⟹ `R−r`; concave (root) ⟹ `R+r`. **This is M4's
  intact-boss survivor torus, reached from the arm side** — the same `geom.Torus` + M4 tessellation
  are reused verbatim. B3 check: `R=50, r=10`, convex ⟹ torus `Center(0,0,90)` axis `ẑ` `(40,10)` —
  bit-identical to OCCT's BREP `5 0 0 90 0 0 1 … 40 10`.
- **Config (ii) — axis ∥ plane (`|s|≈0`), edge = line.** `γ` is a ruling ⟹ the arm is an **exact
  `geom.Cylinder`** of radius `r` about that ruling — the *same* rolling-ball cylinder the Plane∧Plane
  arm already builds, only the spine's provenance differs. B3 check: OCCT emits `2 … 10`.
- **Config (iii) — oblique (`0<|s|<1`), edge = ellipse.** `γ` is an ellipse ⟹ a genuine canal/BSpline
  surface. **Out of Slice A** → honest-reject (keeps the current do-no-harm behaviour).

**The corner (equal radii) is an analytic `geom.Sphere` of radius `r`** — OCCT BREP `4 … 10`, NOT a
BSpline (correcting the trihedral spike's guess). The ball at the vertex is tangent to the cylinder
and the two planes simultaneously; its centre is the common ball-centre where the three arm tubes
meet. So the corner reuses the existing `geom.Sphere` + sphere-patch machinery — only the
**centre solve** generalises from "tangent to 3 planes" to "tangent to cylinder + 2 planes."

**Existence / sign** (honest-reject guards): config (i) convex needs `ρ = R−r > 0` ⟹ `r < R` (at
`r≥R` the tube self-intersects); config (ii) needs `P_r` to actually cut `C_ρ`. The convex/concave
sign is read from the **material side** at the edge, never hard-coded.

## Architecture

The fillet pipeline is unchanged in shape (`computeCorners` → `computeFillets` → assemble/rebuild);
Slice A adds two analytic emitters behind the existing reject sites, cloning the proven do-no-harm
strangler pattern. The hardened assembly/orient/weld layer is untouched.

### Module layout
- `kernel/ops/fillet_curved.go` (modify) — the arm reject site. `curvedFilletError` is replaced (for
  configs i/ii on a Plane∧Cylinder edge) by a dispatch to the new arm builder; configs iii and any
  unfitting geometry fall through to the **unchanged honest reject** (do-no-harm).
- `kernel/ops/fillet_curved_arm.go` (new) — `classifyCurvedArm(edge, cyl, plane) → armKind` (torus /
  cylinder / rejected, by `s = â·n̂_P` with a model-relative angular band) and the two exact
  constructors: `torusArmSurface(cyl, plane, r, side) → geom.Torus` and
  `cylinderArmSurface(edge, r) → geom.Cylinder`, plus the arm's contact rails.
- `kernel/ops/fillet_arm_section.go` (modify) — a plane∧cylinder sibling of `armSectionArc`: the
  cross-section quarter-arc `[φ_P, φ_C]` stationed on the torus/cylinder arm (the rail the corner
  consumes). The existing Plane∧Plane construction is unchanged.
- `kernel/ops/fillet.go` (modify) — `solveBlend` (`:906–914`): generalise the analytic-sphere corner
  to a curved host (centre = ball tangent to cylinder + 2 planes) when the corner is a valid equal-`r`
  sphere; otherwise the current `"corner face must be planar"` reject stands (do-no-harm). The planar
  path stays byte-identical — the curved branch is gated on **a host face being curved**, never on the
  trihedral kind alone.
- `kernel/ops/fillet_setback_partition.go` / M4 machinery (reuse) — the corner setback trim (clip the
  arm's `u`-extent where the sphere takes over) reuses the M4 σ-partition + its `ΣΔ=2π` closure guard.

### The strangler seam (do-no-harm, cloned from the green path)
The arm emitter and the curved-corner both follow `sphereSurfaceViaRail` (`fillet_faces.go:501`): try
the exact analytic build; on any classification/existence/certificate decline, **return the current
reject** so the case rides do-no-harm baseline exactly as today. No case regresses; only the
axis-aligned family greens.

### Ordering (mandatory)
Arm builder lands and is gated **before** the corner. A corner-only change greens nothing (proven).
The arm has its own gate (a case where OCCT asks only the arm, or the B3 body once the corner also
lands); the corner gate is the whole-body area.

## Scope (Slice A only)

- **In:** axis-aligned Plane∧Cylinder arms — config (i) torus + config (ii) cylinder — and the
  equal-radius sphere corner, for a single Cylinder host with two planes at a valence-3 vertex.
  Greens B3/N1/O1 and the `[Cylinder,Plane,Plane]` family (B7/H7/L8/M5/N7…).
- **Out (firewalled):** config (iii) oblique/ellipse edge → general canal surface (**Slice B**);
  cone/sphere/torus/EllipticalCylinder hosts (**their arms as they come online**); cyl∧cyl two-curved-
  host corners O4/O9/P7 and variable radius (**Slice C**). All ride the unchanged honest reject.

## Verification & oracle gates

- **Per-surface faithfulness (topology, not just area — the M3/S7 lesson):** the arm torus is ONE
  intact `geom.Torus` face of area matching `major=R∓r, minor=r` (e.g. B3 the quarter tube), the arm
  cylinder ONE `geom.Cylinder(r)` face, the corner ONE `geom.Sphere(r)` face — asserted by
  `countSurfaceFacesNear[T]` (the existing per-type gate), so a wrong-sign or split surface fails even
  if the total area is coincidentally near.
- **Whole-body area oracle (`deps=1%`):** B3 = 20559.5, N1 = 58091.9, O1 = 65104.9 (next tier B7
  43467.9, L8 61663.5, M5 61187.1, N7 61222.9), via `TestOCCTBlendSimple`.
- **Manifold / volume regression, never `IsSolid` alone** (the convex/concave-sign guard): the result
  is a watertight solid AND its tessellated volume matches the analytic/OCCT value — a wrong-sign torus
  welds inside-out and would pass `IsSolid` but fail volume.
- **Corpus non-regression:** `TestOCCTBlendSimple` rises from 54 by exactly the greened cases; all
  other grids byte-identical (the curved branch is gated on a curved host; planar paths untouched).
- **DRAWEXE oracle** per greened case; env `test-utilities/occt-blend/oracle/drawenv.sh`,
  `printf 'source X.tcl\n' | DRAWEXE -b`.
- **Live test (pre-PR only; this milestone opens no PR):** MCP-bridge fillet of a quarter-cylinder
  (a B3-like body) + screenshot confirming the rounded rim/wall/corner render clean and watertight.

## Risks & pitfalls (from the derivation)

- **Convex/concave sign of the torus major radius (`R−r` vs `R+r`) — TOP risk.** From the material
  side at the edge, never hard-coded; gated by the area oracle + the manifold/volume regression.
- **Config classification near thresholds.** `s = â·n̂_P` is a combinatorial branch — guard with an
  **angular** band scaled to the edge's sampled deviation, not a value constant; cross-check `ρ`
  against the fitted spine radius. The current hard-coded `1e-6` in `curvedFilletError:45` becomes
  `res.Weld()`-relative (ADR-0042).
- **Existence.** `r≥R` (config i) or `P_r` clearing `C_ρ` (config ii) → honest-reject, never a
  self-intersecting/absent arm.
- **Torus seam / frame alignment.** Build the arm torus with a refHint from the neighbouring cyl/cap
  frame (the Oblikovati#129 fix M4 relies on) so the `u=0` seam lines up and the mesh welds watertight.
- **Corner setback extent.** The arm `u`-clip must leave exactly the sphere's span — reuse M4's
  `ΣΔ=2π` closure guard so a wrong traversal sense fails loudly, not a silent overlap/crack.
- **G1 at the corner rail is exact (shared ball centre)** — verify by asserting the arm normal and
  sphere normal agree along the shared quarter-arc, not a tolerance fudge. The F2 ribbon-sign landmine
  does not apply (no obstacle-class geometry in these cases).

## References
- `.superpowers/sdd/m5-curved-arm-derivation.md` (geometry: spine, torus/cylinder KPart, sign,
  existence, composition, pitfalls) — the authoritative geometry source.
- `.superpowers/sdd/m5-trihedral-spike.md` (the masked-dependency proof, the wiring seams, the RailLoop
  the corner needs, the case list + gates).
- `.superpowers/sdd/m4-rim-partition-derivation.md` (the same torus + the σ-partition setback trim).
- OCCT `ChFi3d`/`ChFiDS`/`BRepBlend` KPart recognition (the oracle: Torus/Cylinder/Sphere analytic
  blends before any BSpline — exactly the B3 BREP dump). Ground truth = the DRAWEXE area oracle.
