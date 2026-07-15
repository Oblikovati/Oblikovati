# Curved-Arm Fillet — Slice A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Build the constant-radius rolling-ball arm fillet on an **axis-aligned Plane∧Cylinder edge** (exact `geom.Torus` for a circular edge, exact `geom.Cylinder` for a linear edge) and the equal-radius trihedral corner it feeds (an analytic `geom.Sphere`), greening the axis-aligned `[Cylinder,Plane,Plane]` family — **B3 (20559.5), N1 (58091.9), O1 (65104.9)** and siblings — to the OCCT area oracle (`deps=1%`), topology-faithful and watertight.

**Architecture:** Two exact analytic emitters land behind the existing reject sites, arm-first. The arm builder replaces `curvedFilletError` (`fillet_curved.go`) for configs (i)/(ii); the corner generalises the existing analytic-sphere center solve (`solveBlend`, `fillet.go`) from 3-planes to cylinder+2-planes; the arm↔corner setback trim reuses M4's σ-partition. Every new route clones the `sphereSurfaceViaRail` do-no-harm strangler — nothing regresses, only the axis-aligned family greens. Oblique/elliptical edges (config iii, a genuine canal surface) fall through to the unchanged honest reject → Slice B.

**Tech Stack:** Go (GPL-2.0-only module `oblikovati`), `kernel/ops` B-rep fillet + tessellation; DRAWEXE 8.0.0 oracle.

**Design sources (READ before implementing):**
- `docs/superpowers/specs/2026-07-15-curved-arm-fillet-slice-a-design.md` — the design.
- `.superpowers/sdd/m5-curved-arm-derivation.md` — the geometry-math-advisor derivation. **Load-bearing:** §D1 (spine `γ = P_r ∩ C_ρ`), §D2 (config i → torus `major=R∓r, minor=r`), §D3 (config ii → cylinder `r`), §D5 (composing arm with corner: the cross-section quarter-arc rail, G1 exact via shared ball centre), §Numerical pitfalls (the convex/concave sign, classification band, existence, seam frame).
- `.superpowers/sdd/m5-trihedral-spike.md` — the wiring seams (§3), the RailLoop the corner needs (§2), the case list + gates (§1), the arm-first ordering + landmines (§5).

## Global Constraints

- **NO PR until the whole corpus is green.** Accumulate + commit per task on `feat/occt-blend-parity-corpus`.
- **Arm-before-corner ordering is mandatory** — a corner-only change greens nothing (proven, `m5-trihedral-spike.md`). The arm builder (Tasks 1–3) lands and is unit-gated before the corner (Tasks 4–5).
- **Corpus non-regression, EVERY task:** `go test ./model/feature -run TestOCCTBlendSimple -v 2>&1 | grep -cE '^\s*--- PASS: TestOCCTBlendSimple/'` stays **≥ 54** (the `-v` is REQUIRED). Tasks 1–4 keep it 54 (additive; B3 stays red at a later stage); Task 5 greens B3/N1/O1 (→ 57+); only curved-host cases may move.
- **Keep planar paths byte-identical** — the curved branch is gated on **a host face being curved** (an arm surface / picked edge is Plane∧Cylinder), never on the trihedral kind alone. Planar trihedral/miter/obstacle paths must not move (diff the corpus name set).
- **Do-no-harm backstop** — every new route (arm AND corner) falls back to the current honest reject on any classification/existence/certificate decline, exactly like `sphereSurfaceViaRail` (`fillet_faces.go:501`). Never emit a self-intersecting or absent surface.
- **Topological faithfulness gates (the M3/S7 lesson):** assert ONE intact face per surface type by concrete type + tight area band (`countSurfaceFacesNear[T]`), not whole-body area alone — a wrong-sign/split surface can be area-coincidental.
- **Manifold/VOLUME regression, never `IsSolid` alone** — the convex/concave-sign guard. A wrong-sign torus welds inside-out and passes `IsSolid` but fails volume.
- **Oracle gates (`deps=1%`, do NOT loosen):** B3 = 20559.5, N1 = 58091.9, O1 = 65104.9 (next tier B7 43467.9, L8 61663.5, M5 61187.1, N7 61222.9). DRAWEXE `../occt-build/lin64/gcc/bin/DRAWEXE`, env `test-utilities/occt-blend/oracle/drawenv.sh`, `printf 'source X.tcl\n' | DRAWEXE -b`.
- **Style:** funcs 4–20 lines; files < 500; explicit types; early returns; ≤ 2 indent; no duplication; error/reject messages carry the offending value + expected shape.
- **Tolerances model-relative** (`ResolutionForPoints(...).Weld()`, ADR-0042); NO bare `1e-6` — the existing `1e-6` in `curvedFilletError:45` becomes `res.Weld()`-relative. Angular tol `ε_ang ≈ k·res.Weld()/ρ`, `k≈2..4`.
- **SPDX** `// SPDX-License-Identifier: GPL-2.0-only` on every new `.go`. Torus arm built with the neighbouring-frame refHint (`NewTorusWithRef`, the Oblikovati#129 seam-alignment fix M4 relies on).

## File Structure

- `kernel/ops/fillet_curved_arm.go` (new) — `classifyCurvedArm` (config i/ii/reject by `s=â·n̂_P`), `torusArmSurface`/`cylinderArmSurface` exact constructors + their contact rails + the material-side sign.
- `kernel/ops/fillet_arm_section.go` (modify) — a Plane∧Cylinder sibling of `armSectionArc`: the cross-section quarter-arc `[φ_P, φ_C]` stationed on the torus/cylinder arm.
- `kernel/ops/fillet_curved.go` (modify) — `computeEdgeFillet`'s Plane∧Cylinder branch: dispatch to the arm builder (config i/ii), fall through to the unchanged `curvedFilletError` (config iii + declines).
- `kernel/ops/fillet.go` (modify) — `solveBlend` (`:906–914`): the curved-host analytic-sphere corner (center = ball tangent to cylinder + 2 planes); relax the `"corner face must be planar"` reject only for a valid equal-`r` sphere corner, do-no-harm otherwise.
- `kernel/ops/fillet_setback_partition.go` / M4 machinery (reuse) — the arm↔corner setback trim + `ΣΔ=2π` closure guard.
- Tests: `kernel/ops/fillet_curved_arm_test.go` (new), `kernel/ops/fillet_curved_test.go` (modify), `kernel/ops/fillet_setback_close_test.go` (faithfulness), `model/feature/occtparity/*` (the B3/N1/O1 area gate is `TestOCCTBlendSimple`).

---

## Task 1: Curved-arm edge classifier

**Problem:** the arm builder must branch on the edge's config by the axis–plane angle `s = â·n̂_P`, with a model-relative angular band (a combinatorial branch — never a bare constant; §Numerical pitfalls).

**Files:** Create `kernel/ops/fillet_curved_arm.go`; test `kernel/ops/fillet_curved_arm_test.go`.

**Interfaces:**
- Consumes: the picked `*topo.Edge`, its adjacent `geom.Cylinder` + `geom.Plane` (as `cylinderPlaneEdge` already extracts, `fillet_curved.go`), `ResolutionForPoints(...).Weld()`.
- Produces: `type armKind int` (`armTorus`, `armCylinder`, `armRejected`); `classifyCurvedArm(cyl geom.Cylinder, pl geom.Plane, res Resolution) armKind`.

- [ ] **Step 1: Write the failing test** — `kernel/ops/fillet_curved_arm_test.go`:

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import "testing"

func TestClassifyCurvedArm(t *testing.T) {
	res := testResolution() // a model-scaled Resolution ~ the corpus body (mimic an existing test helper)
	// axis ⊥ plane (|s|=1) → torus arm (B3 top-rim edge)
	if k := classifyCurvedArm(cylAxis(0, 0, 1, 50), planeNormal(0, 0, 1), res); k != armTorus {
		t.Fatalf("axis ⊥ plane: want armTorus, got %v", k)
	}
	// axis ∥ plane (s=0) → cylinder arm (B3 vertical-wall edge)
	if k := classifyCurvedArm(cylAxis(0, 0, 1, 50), planeNormal(1, 0, 0), res); k != armCylinder {
		t.Fatalf("axis ∥ plane: want armCylinder, got %v", k)
	}
	// oblique → rejected (Slice B)
	if k := classifyCurvedArm(cylAxis(0, 0, 1, 50), planeNormal(0, 0.6, 0.8), res); k != armRejected {
		t.Fatalf("oblique: want armRejected, got %v", k)
	}
}
```
(Use/extend the existing `geom.Cylinder`/`geom.Plane` test constructors — grep how `fillet_curved_test.go` builds them; add `cylAxis`/`planeNormal`/`testResolution` helpers if none fit.)

- [ ] **Step 2: Run → FAIL** (`classifyCurvedArm` undefined). `go test ./kernel/ops -run TestClassifyCurvedArm -v`.

- [ ] **Step 3: Implement `classifyCurvedArm`** — compute `s = â·n̂_P`; `|s| > 1 − ε_ang` → `armTorus`; `|s| < ε_ang` → `armCylinder`; else `armRejected`. `ε_ang = k·res.Weld()/R` (§Numerical pitfalls). Funcs 4–20 lines, SPDX header.

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Corpus unchanged** — `TestOCCTBlendSimple` = 54 (classifier unwired). Build/vet/lint clean.

- [ ] **Step 6: Commit** — `feat(blend): curved-arm edge classifier (torus/cylinder/reject by axis-plane angle)`.

---

## Task 2: Exact arm surface constructors + section arc

**Problem:** emit the exact analytic arm surface (derivation §D2/§D3) and the cross-section quarter-arc the corner consumes (§D5).

**Files:** Modify `kernel/ops/fillet_curved_arm.go` + `kernel/ops/fillet_arm_section.go`; test `kernel/ops/fillet_curved_arm_test.go`.

**Interfaces:**
- Produces: `torusArmSurface(cyl geom.Cylinder, pl geom.Plane, r float64, res Resolution) (geom.Torus, bool)` (major `R∓r` by material side, minor `r`, center = axis point projected onto `P_r`, seam refHint from the cyl frame via `NewTorusWithRef`); `cylinderArmSurface(edge *topo.Edge, cyl geom.Cylinder, pl geom.Plane, r float64) (geom.Cylinder, bool)` (radius `r` about the selected `P_r ∩ C_ρ` ruling); `curvedArmSectionArc(arm geom.Surface, station float64) geom.Curve3` (the `[φ_P, φ_C]` quarter-arc — sibling of `armSectionArc`).

- [ ] **Step 1: Write the failing tests** — assert the EXACT B3 values (derivation §D2 check):

```go
func TestTorusArmSurface_B3(t *testing.T) {
	res := testResolution()
	tor, ok := torusArmSurface(cylAxisAt(0, 0, 0, /*axis*/ 0, 0, 1, /*R*/ 50), planeAtZ(100), /*r*/ 10, res)
	if !ok {
		t.Fatalf("torusArmSurface declined a valid convex B3 rim")
	}
	// B3: convex ⟹ major = R−r = 40, minor = r = 10, center z = 100−10 = 90, axis ẑ
	if !nearly(tor.MajorRadius, 40) || !nearly(tor.MinorRadius, 10) || !nearly(tor.Center.Z, 90) {
		t.Fatalf("B3 torus arm = {major %.3f, minor %.3f, cz %.3f}, want {40,10,90} (OCCT BREP 5 0 0 90 … 40 10)",
			tor.MajorRadius, tor.MinorRadius, tor.Center.Z)
	}
}

func TestCylinderArmSurface_B3(t *testing.T) {
	res := testResolution()
	cyl, ok := cylinderArmSurface(b3VerticalWallEdge(t), cylAxisAt(0, 0, 0, 0, 0, 1, 50), planeNormal(1, 0, 0), 10)
	if !ok || !nearly(cyl.Radius, 10) {
		t.Fatalf("B3 cylinder arm radius = %.3f (ok=%v), want 10", cyl.Radius, ok)
	}
}
```

- [ ] **Step 2: Run → FAIL** (constructors undefined).

- [ ] **Step 3: Implement** `torusArmSurface` (§D2: material-side sign → `ρ=R∓r`, center = `A` projected onto `P_r` at offset `r` along `â`, `NewTorusWithRef` refHint), `cylinderArmSurface` (§D3: the selected ruling of `P_r ∩ C_ρ`, radius `r`), and `curvedArmSectionArc` (§D5: the quarter-circle `[φ_P,φ_C]` on the tube at the given station). Existence guards: `torusArmSurface` returns `false` if `ρ=R−r < k·res.Weld()` (convex) — honest-reject; `cylinderArmSurface` returns `false` if `P_r` clears `C_ρ`. Funcs 4–20 lines.

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Corpus unchanged** — 54 (still unwired). Build/vet/lint clean.

- [ ] **Step 6: Commit** — `feat(blend): exact torus/cylinder arm surfaces + section arc (Plane∧Cylinder)`.

---

## Task 3: Wire the arm builder into `computeEdgeFillet`

**Problem:** replace the `curvedFilletError` reject (config i/ii) with the arm builder; config iii + any decline fall through to the unchanged reject (do-no-harm). Tested via a direct `computeEdgeFillet` call (the corner reject masks the full pipeline until Task 4).

**Files:** Modify `kernel/ops/fillet_curved.go`; test `kernel/ops/fillet_curved_test.go`.

**Interfaces:** Consumes `classifyCurvedArm`/`torusArmSurface`/`cylinderArmSurface`/`curvedArmSectionArc`. The arm surface + section arc populate the same `edgeFillet` the straight-edge path produces (grep `computeEdgeFillet`'s return + the `edgeFillet` struct so the torus/cylinder arm slots into the existing rebuild).

- [ ] **Step 1: Write the failing test** — call `computeEdgeFillet` directly on B3's Plane∧Cylinder top-rim edge; assert it returns an `edgeFillet` carrying the torus arm, NOT an error:

```go
func TestComputeEdgeFillet_B3TorusArm(t *testing.T) {
	body := importCorpusSolid(t, "simple/B3")
	e := b3TopRimEdge(t, body) // the Cyl∧Plane circular edge at the top rim
	ef, err := computeEdgeFillet(body, filletPick{edge: e, radius: 10} /*match the real signature*/)
	if err != nil {
		t.Fatalf("computeEdgeFillet on B3 curved rim errored (arm not built): %v", err)
	}
	if _, ok := ef.surface.(geom.Torus); !ok { // adjust to the edgeFillet's surface accessor
		t.Fatalf("B3 curved rim arm = %T, want geom.Torus", ef.surface)
	}
}
```

- [ ] **Step 2: Run → FAIL** (`curvedFilletError`).

- [ ] **Step 3: Wire it** — in `computeEdgeFillet`'s `cylinderPlaneEdge` branch, `classifyCurvedArm`; `armTorus`→`torusArmSurface`, `armCylinder`→`cylinderArmSurface`, both populating the `edgeFillet`; `armRejected` or a `false` constructor → the existing `curvedFilletError` (do-no-harm). Make `curvedFilletError`'s tangency `1e-6` `res.Weld()`-relative while here.

- [ ] **Step 4: Run → PASS** (torus arm built).

- [ ] **Step 5: Corpus non-regression + any bonus greens** — `TestOCCTBlendSimple` **≥ 54**, byte-identical on planar cases (diff the name set). B3 stays red (corner still rejects) — confirm it fails at the CORNER now, not the arm. If a pure-arm curved red (a Plane∧Cylinder edge with no curved corner) greens here, that's a welcome bonus — record it, confirm its area matches OCCT. Build/vet/lint clean.

- [ ] **Step 6: Commit** — `feat(blend): build the axis-aligned Plane∧Cylinder arm fillet`.

---

## Task 4: Curved-host analytic-sphere corner

**Problem:** the trihedral corner over a curved host is an analytic `geom.Sphere(r)` (advisor BREP code 4) whose center is the ball tangent to the cylinder + 2 planes. Generalise `solveBlend`'s center solve and relax the `"corner face must be planar"` reject (`fillet.go:906–914`) for a valid equal-`r` sphere; do-no-harm otherwise. Reuse the existing sphere-patch machinery (`spherePatchFace`/`sphereSurfaceViaRail`) unchanged.

**Files:** Modify `kernel/ops/fillet.go` (`solveBlend`); test `kernel/ops/fillet_curved_test.go`.

**Interfaces:** Produces a `cornerBlend` carrying `geom.Sphere{Center: c, Radius: r}` for a curved-host trihedral corner, where `c` is the ball-center tangent to the cylinder (distance `R∓r` from its axis) and the 2 planes (distance `r` from each). Consumed by the unchanged `computeFillets`/rebuild.

- [ ] **Step 1: Write the failing test** — `solveBlend`/`computeCorners` on B3's `[Cylinder,Plane,Plane]` corner returns a `geom.Sphere` near OCCT's `(38.73, 10, 90) r=10`, not the reject:

```go
func TestSolveBlend_B3CurvedCorner(t *testing.T) {
	body := importCorpusSolid(t, "simple/B3")
	blends, _, err := computeCorners(b3Picks(t, body)) // the 3 picks, radius 10
	if err != nil {
		t.Fatalf("computeCorners on B3 curved corner errored (still rejecting curved host): %v", err)
	}
	cb := b3Corner(blends) // the single valence-3 corner
	sph, ok := cb.sphere.(geom.Sphere)
	if !ok || !nearly(sph.Radius, 10) || !nearlyPt(sph.Center, math.P3(38.73, 10, 90)) {
		t.Fatalf("B3 corner sphere = %+v (ok=%v), want center (38.73,10,90) r10 (OCCT BREP 4)", cb.sphere, ok)
	}
}
```

- [ ] **Step 2: Run → FAIL** (`"corner face must be planar"`).

- [ ] **Step 3: Implement** — in `solveBlend`, when a host face is a `geom.Cylinder` (not plane): compute the ball center tangent to the cylinder (`|c − axis-projection| = R∓r`, same material-side sign as the arm) + the 2 planes (`(c−p_P)·n̂_P = −r`), build `geom.Sphere{c, r}`; verify the equal-`r` corner is valid (all three arm radii equal, center consistent) else return the existing reject (do-no-harm). Gate the new branch on **a host face being curved**, keeping the all-planar path byte-identical.

- [ ] **Step 4: Run → PASS** (sphere center matches OCCT).

- [ ] **Step 5: Corpus non-regression** — `TestOCCTBlendSimple` **≥ 54**, planar cases byte-identical. B3 may still be red (corner builds but the arm↔sphere weld needs the Task-5 setback) — confirm it now fails at assembly/solid, not at the corner reject. Build/vet/lint clean.

- [ ] **Step 6: Commit** — `feat(blend): analytic-sphere corner over a curved (cylinder) host`.

---

## Task 5: Arm↔corner setback trim + weld → green B3/N1/O1

**Problem:** clip each arm's `u`-extent where the corner sphere takes over (reuse M4's σ-partition + `ΣΔ=2π` closure guard) and weld the arms + sphere into a watertight solid, greening B3/N1/O1 end-to-end.

**Files:** Modify the assembly/setback path (reuse `kernel/ops/fillet_setback_partition.go` + M4 machinery; the arm↔corner trim is the same σ-ruler clip); tests `kernel/ops/fillet_setback_close_test.go` (faithfulness) + `model/feature/occtparity` (the area gate is `TestOCCTBlendSimple`).

- [ ] **Step 1: Write the failing faithfulness + area tests** — `go test ./model/feature -run 'TestOCCTBlendSimple/B3$' -v` → FAIL. Add:

```go
func TestFilletEdges_B3CurvedArmIntact(t *testing.T) {
	body := filletedCorpusEdges(t, "simple/B3", 10) // all 3 picks; helper sibling of filletedCorpusEdge
	if !body.IsSolid() {
		t.Fatalf("B3 curved-arm fillet is not a solid: %d open edges", len(openEdges(body)))
	}
	if r := Validate(body); !r.Valid || !r.HolesContained {
		t.Fatalf("B3 curved-arm fillet invalid: Valid=%v HolesContained=%v", r.Valid, r.HolesContained)
	}
	// one intact torus arm (major=R−r=40, minor=r=10 quarter tube), one cylinder arm (r=10),
	// one sphere corner (r=10) — topology-faithful, not area-coincidental.
	if got := countSurfaceFacesNear[geom.Torus](body, torusQuadrantArea(40, 10), 5); got != 1 {
		t.Fatalf("B3: want ONE intact torus arm (major40/minor10), got %d", got)
	}
	if got := countSurfaceFacesNear[geom.Sphere](body, sphereOctantArea(10), 2); got != 1 {
		t.Fatalf("B3: want ONE analytic sphere corner (r10), got %d", got)
	}
}
```
(Compute `torusQuadrantArea`/`sphereOctantArea` from the trimmed extents, or read the OCCT per-face areas via DRAWEXE `sprops` on each arm/corner face and assert against those exact values with a ~1% band.)

- [ ] **Step 2: Run → FAIL** (assembly not watertight / area off).

- [ ] **Step 3: Implement the setback trim + weld** — clip each arm's `u`-extent at the corner station (the σ-partition footprint-line cut, `m4-rim-partition-derivation.md` §D2, guarded by `ΣΔ=2π`); weld arm↔arm and arm↔sphere along their shared quarter-arcs (matched sampling, as M4). Honest-reject the whole corner to baseline on any closure/weld decline. Verify G1 by asserting arm normal == sphere normal along the shared arc (§Numerical pitfalls), not a fudge.

- [ ] **Step 4: Run → PASS** — B3 within **[20353.9, 20765.1]** ([20559.5·0.99, ·1.01]); `TestFilletEdges_B3CurvedArmIntact` green.

- [ ] **Step 5: N1 + O1 green** — `TestOCCTBlendSimple/N1$` within 1% of **58091.9**, `/O1$` within 1% of **65104.9** (same geometry, radius 5). Add their faithfulness asserts if they differ structurally; else the area gate suffices. **Manifold/volume regression** on B3/N1/O1 (tessellated volume matches OCCT — the wrong-sign guard).

- [ ] **Step 6: Corpus** — `TestOCCTBlendSimple` 54 → **57** (B3/N1/O1), planar cases byte-identical, other grids no new failures. Build/vet/lint clean.

- [ ] **Step 7: Commit** — `feat(blend): green B3/N1/O1 (axis-aligned curved-arm trihedral fillet)`.

---

## Task 6: Extend to the axis-aligned family + milestone verification

**Files:** the gate is `TestOCCTBlendSimple`; add faithfulness asserts only where a case differs structurally.

- [ ] **Step 1: Sweep the remaining axis-aligned `[Cylinder,Plane,Plane]` cases** — B7 (43467.9, r10), L8 (61663.5, r5), M5 (61187.1, r5), N7 (61222.9, r5), H7 (554732, r10). Each should green with no new code (same geometry); for any that don't, diagnose against its DRAWEXE oracle (a different config, a concave arm, an existence edge-case) — fix within the do-no-harm envelope or honest-reject + document if it's actually a later slice (e.g. a hidden oblique edge).
- [ ] **Step 2: Whole-milestone verification** — full corpus count, all M4 tessellation tripwires still green (the arm torus reuses M4's tessellation — confirm `TestP2TorusBandNotFullDomain`, `TestRimFilletTorusBand` unaffected), `go build ./kernel/... && go vet && golangci-lint run ./kernel/ops/...` clean. Diff the corpus name set vs base to confirm only curved-host axis-aligned cases moved.
- [ ] **Step 3: Commit** — `feat(blend): axis-aligned curved-arm family + Slice A verification`.

---

## Live test (pre-PR only — this milestone opens NO PR)

Per CLAUDE.md, before the eventual whole-corpus PR: drive `Oblikovati.AddIns.MCPBridge` to fillet a quarter-cylinder (a B3-like body) on its rim/wall/radial edges, `Recompute`, MCP-screenshot — confirm the rounded rim (torus), wall (cylinder), and corner (sphere) render clean, watertight, no inside-out shading. This milestone opens no PR (the full corpus is not green).

## Notes for the executor

- **Read order per task:** Tasks 1–3 → derivation §D2/§D3 + spike §2/§3. Task 4 → §D5 (composition) + spike §3 (the `solveBlend` seam). Task 5 → §D5 (setback) + `m4-rim-partition-derivation.md` §D2/§D3 (the σ-partition clip + closure guard).
- **Arm-first, no shortcut:** the corner (Task 4) is untestable end-to-end until its arms (Task 3) exist; that's why the arm is unit-tested via a direct `computeEdgeFillet` call. Tasks 1–4 keep corpus 54; Task 5 flips B3/N1/O1 green with arm + corner + setback all in place.
- **The convex/concave sign is the top risk** — from the material side, gated by the manifold/volume regression, never `IsSolid` alone. If B3's torus area comes out as the R+r (major 60) torus, the sign is flipped.
- **Honest-reject over fabrication** — any config-iii edge, non-fitting geometry, or failed weld returns the current reject; never a self-intersecting torus or an overlapping corner.
- **Keep planar byte-identical** — the curved branch gates on a curved host/edge; the all-planar trihedral/miter path must not move (diff the name set every task).
