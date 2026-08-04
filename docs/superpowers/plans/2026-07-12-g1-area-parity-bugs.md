# G1 — Area-Parity Fillet Bugs — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the cases where our fillet builds a *valid* solid with the wrong area — fix
where tractable, and for the deferred apex-edge cluster, replace silent-wrong solids with an
honest rejection guarded by a zero-false-positive predicate.

**Architecture:** Three independent clusters (spec
`docs/superpowers/specs/2026-07-12-g1-area-parity-bugs-design.md`). 1a = revolution-axis apex
edge (interim honest-rejection guard; real fix deferred to G5). 1b = box shared-corner
double-count (fix + analytic regression). 1c = restored solids (per-case triage). Work is in
`kernel/ops/` (the fillet engine) with the `occtparity` corpus as the gate.

**Tech Stack:** Go, `kernel/ops` fillet engine, `kernel/geom` (Line/Cylinder/Cone/Sphere),
`kernel/topo`, the `occtparity` corpus (`model/feature/occtparity`).

## Global Constraints

- **Never loosen the gate.** No relaxing the 1% area assertion; no skipping a case OCCT runs.
- **No silent wrong solids.** Floor for every 1a case is an honest `Health` rejection with a
  specific reason. Shipping a valid-but-wrong solid is the exact defect being removed.
- **No result-plausibility heuristics** (no "removed too much volume"). Rejection is on a
  structural input predicate or a tangency correctness invariant only.
- **Zero-false-positive gate is mandatory** for any guard: it must fire on none of the passing
  fillets (the 13 green corpus cases + the `kernel/ops` and `model/feature` fillet suites).
- **Bug fixes get a kernel-level regression test.** Root-cause with minimal `kernel/ops`
  reproductions; validate corpus outcomes through the real feature path.
- **SPDX** `// SPDX-License-Identifier: GPL-2.0-only` on every new `.go`; functions ≤20 lines;
  explicit types; `oblikovati.org/math` (`stdmath "math"` where both needed).
- Grounding: hard corner/tangency geometry → consult the `geometry-math-advisor` skill; OCCT
  `ChFi3d` is the oracle.

## Verified facts (from investigation — do not re-derive)

- 1a picked edge is the revolution-**axis apex** edge: straight, both neighbour faces are
  `geom.Plane` (the two radial cut faces), endpoints valence-3 incident only to planes. The
  distinguishing invariant: **both radial faces each border the same quadric-of-revolution
  face**, and the edge line is that quadric's axis (cylinder: `(Origin,AxisDir)`; cone: through
  `Apex` dir `AxisDir`; sphere: passes through `Center`).
- A9 (90°, apex **convex**) and B4 (270°, apex **concave**) produce byte-identical output
  (area 19098.9, vol 122853.2) — the fillet ignores the dihedral. 1a cases:
  `simple/{A9,B4,B8,C3,D2,D6}`.
- 1b: `simple/{P8,V8}` `box 5 5 5` + two r=1 fillets on corner-sharing edges → area +3.4%.
- 1c: `simple/{W2,Y2,Q1,H6}`.
- Geometry APIs: `edge.Geometry() geom.Curve3`; a straight edge's curve is `geom.Line{Origin
  math.Point3; Dir math.UnitVector3}`. `face.Geometry() geom.Surface`; `face.Edges() []*Edge`.
  `geom.Cylinder{Origin,AxisDir,Radius}`, `geom.Cone{Apex,AxisDir,HalfAngle}`,
  `geom.Sphere{Center,Radius}`. `ops.ClassifyEdgeConvexity(e)`, `ops.FilletEdges(body, keys, r)`.
- STEP fixtures for corpus cases live under `model/feature/occtparity/fixtures/<grid>/`. To get
  an oriented input solid for a kernel test, generate it with the oracle:
  `test-utilities/occt-blend/oracle/occt_blend_oracle ../OCCT/tests/blend/simple/A9 <outdir>`.

---

## Phase 1 — Cluster 1a: root-cause + interim guard

### Task 1: Characterize the apex-fillet failure and freeze the fix-vs-guard decision

**Files:**
- Create: `kernel/ops/fillet_apex_diagnosis_test.go`

**Interfaces:**
- Produces: a committed diagnosis (in the test's doc comment) of *why* the apex fillet is wrong
  and whether a trivial fix exists; a failing test pinning A9/B4 current wrong behavior.

- [ ] **Step 1: Write a characterization test** that imports A9's and B4's input (oracle STEP),
  fillets the located apex edge r=10 via `ops.FilletEdges`, and records result area+volume+face
  count. Assert the *correct* (OCCT) areas so the test FAILS, documenting current output.

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops_test
// TestApexFilletA9B4 pins the 1a defect: filleting the revolution-axis apex edge of a partial
// pcylinder. A9 (90°, convex apex) and B4 (270°, concave apex) currently yield byte-identical
// wrong geometry; OCCT expects 21308.8 and 44956.6 respectively. This test drives root-cause
// and MUST fail until 1a is fixed or (interim) the apex fillet is honestly rejected.
func TestApexFilletA9B4(t *testing.T) { /* import fixture, FilletEdges apex r10, assert OCCT area */ }
```

- [ ] **Step 2: Run it, confirm it fails**, capturing the actual built geometry.

Run: `go test ./kernel/ops/ -run TestApexFilletA9B4 -v`
Expected: FAIL with our wrong areas (≈19098.9 both), documenting the defect.

- [ ] **Step 3: Diagnose** — inspect the 6-face result: which corner faces the reconstruction
  builds at the two axis vertices and why it removes ~73000 vol³; whether convex/concave is
  ignored at `filletFrame`/corner assembly. Record the finding + the decision (trivial fix in
  `kernel/ops` vs defer-to-G5) in the test file's doc comment. If a trivial, low-risk fix is
  found, note it for Task 3; otherwise the guard (Task 2/3) is the deliverable.

- [ ] **Step 4: Commit** the diagnosis test.

```bash
git add kernel/ops/fillet_apex_diagnosis_test.go
git commit -m "test(fillet): pin the revolution-axis apex fillet defect (G1 1a A9/B4)"
```

### Task 2: The `revolutionAxisApexEdge` predicate + zero-false-positive gate

**Files:**
- Create: `kernel/ops/fillet_apex_guard.go`
- Create: `kernel/ops/fillet_apex_guard_test.go`

**Interfaces:**
- Produces: `func revolutionAxisApexEdge(e *topo.Edge) bool` — true iff `e` is a straight edge
  whose two planar faces both border a common quadric-of-revolution face and whose line is that
  quadric's axis. Consumed by Task 3.

- [ ] **Step 1: Write the predicate test** — true for A9/B4/B8/C3/D2/D6 apex edges; false for
  every edge of a plain `brep.SolidBlock` box (the primary false-positive risk).

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops
// asserts revolutionAxisApexEdge fires on the 6 apex edges and on NO box edge.
func TestRevolutionAxisApexEdgePredicate(t *testing.T) { /* build/import, assert per edge */ }
```

- [ ] **Step 2: Run — fails (undefined).**

Run: `go test ./kernel/ops/ -run TestRevolutionAxisApexEdgePredicate`
Expected: FAIL — undefined.

- [ ] **Step 3: Implement `fillet_apex_guard.go`** — the structural predicate, small helpers:

```go
// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// revolutionAxisApexEdge reports whether e is the apex edge of a partial revolved primitive:
// a straight edge, both faces planar, both faces bordering one common quadric-of-revolution
// face, and e's line being that quadric's axis. This is the verified structural signature of
// the 1a defect (a fillet here currently ships a silently-wrong solid); it is checked BEFORE
// filleting so it cannot be fooled by the broken result.
func revolutionAxisApexEdge(e *topo.Edge) bool {
	line, ok := e.Geometry().(geom.Line)
	if !ok {
		return false
	}
	faces := e.Faces()
	if len(faces) != 2 || !bothPlanar(faces) {
		return false
	}
	q, ok := commonQuadricNeighbour(faces[0], faces[1])
	if !ok {
		return false
	}
	return lineIsQuadricAxis(line, q)
}

func bothPlanar(faces []*topo.Face) bool {
	for _, f := range faces {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			return false
		}
	}
	return true
}

// commonQuadricNeighbour returns a quadric-of-revolution face that BOTH a and b border.
func commonQuadricNeighbour(a, b *topo.Face) (geom.Surface, bool) {
	for _, fa := range borderingFaces(a) {
		if !isRevolutionQuadric(fa.Geometry()) {
			continue
		}
		for _, fb := range borderingFaces(b) {
			if fb == fa {
				return fa.Geometry(), true
			}
		}
	}
	return nil, false
}

func borderingFaces(f *topo.Face) []*topo.Face {
	var out []*topo.Face
	for _, e := range f.Edges() {
		for _, nf := range e.Faces() {
			if nf != f {
				out = append(out, nf)
			}
		}
	}
	return out
}

func isRevolutionQuadric(s geom.Surface) bool {
	switch s.(type) {
	case geom.Cylinder, geom.Cone, geom.Sphere:
		return true
	}
	return false
}

// lineIsQuadricAxis reports whether line is the quadric's axis of revolution.
func lineIsQuadricAxis(line geom.Line, s geom.Surface) bool {
	const tol = 1e-6
	switch q := s.(type) {
	case geom.Cylinder:
		return collinear(line, q.Origin, q.AxisDir, tol)
	case geom.Cone:
		return collinear(line, q.Apex, q.AxisDir, tol)
	case geom.Sphere:
		return pointOnLine(line, q.Center, tol) // a sphere sector's apex axis passes through the center
	}
	return false
}

// collinear reports whether line runs along axis (origin, dir): same direction and origin on line.
func collinear(line geom.Line, origin math.Point3, dir math.UnitVector3, tol float64) bool {
	if 1-stdAbs(float64(line.Dir.Dot(dir))) > tol {
		return false
	}
	return pointOnLine(line, origin, tol)
}

// pointOnLine reports whether p lies on line within tol (perpendicular distance).
func pointOnLine(line geom.Line, p math.Point3, tol float64) bool {
	v := line.Origin.VectorTo(p)
	perp := v.Sub(line.Dir.AsVector().Scale(v.Dot(line.Dir.AsVector())))
	return float64(perp.Length()) <= tol
}
```
(Confirm the exact `math` accessors — `VectorTo`, `Sub`, `Dot`, `Scale`, `Length` — against
`math/*.go`; add a small `stdAbs` or use `stdmath.Abs`. Keep every function ≤20 lines.)

- [ ] **Step 4: Run the predicate test — passes.**

Run: `go test ./kernel/ops/ -run TestRevolutionAxisApexEdgePredicate -v`
Expected: PASS — fires on the 6 apex edges, no box edge.

- [ ] **Step 5: The zero-false-positive gate** — add a test that runs `revolutionAxisApexEdge`
  over every edge of every currently-PASSING corpus fixture (the 13 green cases) and asserts it
  never fires on them. Place it in `model/feature/occtparity` (it needs the corpus + fixtures).

```go
// SPDX-License-Identifier: GPL-2.0-only
// In model/feature/occtparity: assert the apex guard never fires on a passing fillet.
// (Expose revolutionAxisApexEdge via a thin ops wrapper, or replicate the corpus edge walk.)
func TestApexGuardNoFalsePositiveOnPasses(t *testing.T) { /* for each PASS case, no edge matches */ }
```

- [ ] **Step 6: Run the gate + the whole fillet suite** (the other false-positive oracle).

Run: `go test ./kernel/ops/ -run 'Fillet|Convex|Chamfer' && go test ./model/feature/ -run 'Fillet|Chamfer|TestApexGuardNoFalsePositive'`
Expected: PASS — no previously-valid fillet triggers the predicate.

- [ ] **Step 7: Commit.**

```bash
git add kernel/ops/fillet_apex_guard.go kernel/ops/fillet_apex_guard_test.go model/feature/occtparity/apex_guard_fp_test.go
git commit -m "feat(fillet): revolution-axis apex-edge predicate + zero-false-positive gate (G1 1a)"
```

### Task 3: Wire the guard (honest rejection) into the fillet path

**Files:**
- Modify: `kernel/ops/fillet.go` (`computeEdgeFillet`, near the existing guards)
- Modify: `kernel/ops/fillet_apex_diagnosis_test.go` (flip to assert the honest rejection)

**Interfaces:**
- Consumes: `revolutionAxisApexEdge` (Task 2). Produces: a specific error so the feature
  `Health` reads honestly.

- [ ] **Step 1: Add the guard** at the top of `computeEdgeFillet`, before `edgePlanarFaces`:

```go
if revolutionAxisApexEdge(e) {
	return edgeFillet{}, fmt.Errorf("fillet: rounding the revolution-axis apex edge of a " +
		"partial revolved solid is not yet supported (tracked: G5 corner reconstruction)")
}
```
(If Task 1 found a trivial correct fix, implement that here instead and turn the 6 green — the
guard is the floor, not the ceiling.)

- [ ] **Step 2: Update the diagnosis test** to assert the honest rejection (health sick with the
  reason) rather than the OCCT area — unless Task 1's fix made them green, in which case assert
  the OCCT areas within 1%.

- [ ] **Step 3: Run — the 6 reject honestly (or pass); the corpus scoreboard moves them from
  FAIL(area)/silent-wrong to an honest FAIL(faulty).**

Run: `go test ./kernel/ops/ -run TestApexFilletA9B4 -v && go test ./model/feature/ -run TestOCCTBlendSimple 2>&1 | tail -5`
Expected: A9/B4 honest reject (or pass); no silent wrong solid.

- [ ] **Step 4: Full fillet-suite regression check** (the guard must not change any passing case).

Run: `go test ./kernel/ops/ ./model/feature/ -run 'Fillet|Chamfer|Convex' 2>&1 | tail -3`
Expected: ok.

- [ ] **Step 5: Commit.**

```bash
git add kernel/ops/fillet.go kernel/ops/fillet_apex_diagnosis_test.go
git commit -m "fix(fillet): honestly reject revolution-axis apex fillet instead of silent wrong solid (G1 1a)"
```

---

## Phase 2 — Cluster 1b: box shared-corner double-count

### Task 4: Minimal analytic reproduction of the +3.4% overshoot

**Files:**
- Create: `kernel/ops/fillet_box_corner_test.go`

**Interfaces:**
- Produces: a failing kernel test asserting the *analytic* expected area of `box 5 5 5` with two
  r=1 fillets on corner-sharing edges (OCCT reference 145.137), isolating where the excess lives.

- [ ] **Step 1: Write the test** — `brep.SolidBlock` 5³, fillet two edges meeting at a corner
  (r=1) via `ops.FilletEdges`, assert area 145.137 within 1% (fails at ≈150.1). Add a comment
  deriving the analytic area: 6 faces minus the two edge strips' footprints, plus two
  quarter-cylinder strips, plus one corner sphere-octant patch — so the reviewer can see the
  expected corner geometry.

- [ ] **Step 2: Run — fails at ~+3.4%.**

Run: `go test ./kernel/ops/ -run TestBoxSharedCornerArea -v`
Expected: FAIL (≈150.1 vs 145.137).

- [ ] **Step 3: Diagnose** whether the excess is a double-added corner patch or overlapping
  strips (compare face inventory + per-face areas against the analytic breakdown). Record in the
  test comment.

- [ ] **Step 4: Commit** the failing reproduction.

```bash
git add kernel/ops/fillet_box_corner_test.go
git commit -m "test(fillet): analytic box shared-corner area reproduction (G1 1b, currently red)"
```

### Task 5: Fix the corner double-count

**Files:**
- Modify: `kernel/ops/` corner-assembly code (located in Task 4's diagnosis)

- [ ] **Step 1: Apply the fix** for the identified double-count/overlap. Ground the corner-patch
  geometry in `geometry-math-advisor` if the correction is non-trivial.
- [ ] **Step 2: Run Task 4's test — passes** (145.137 within 1%).

Run: `go test ./kernel/ops/ -run TestBoxSharedCornerArea -v`
Expected: PASS.

- [ ] **Step 3: Corpus + fillet-suite regression.**

Run: `go test ./kernel/ops/ ./model/feature/ -run 'Fillet|Chamfer|TestOCCTBlendSimple' 2>&1 | tail -3`
Expected: P8 and V8 now PASS; nothing else regresses.

- [ ] **Step 4: Commit.**

```bash
git add kernel/ops/
git commit -m "fix(fillet): correct box shared-corner patch double-count (G1 1b, P8/V8 green)"
```

---

## Phase 3 — Cluster 1c: restored solids, per case

### Task 6: Triage W2, Y2, Q1, H6

**Files:**
- Create: `kernel/ops/fillet_restored_1c_test.go` (one sub-test per case)
- Modify: `docs/superpowers/specs/2026-07-11-occt-blend-greening-roadmap.md` (reclassifications)

- [ ] **Step 1:** For each of W2/Y2/Q1/H6, write a sub-test that imports the fixture, fillets the
  located edge, and asserts the OCCT area (fails). Capture each defect (curved neighbour? strip
  self-intersection? radius error?).
- [ ] **Step 2:** Fix the ones that are small and self-contained (assert green). For any that is
  really a different package's defect (e.g. curved-neighbour), **reclassify it in the roadmap**
  (move the case id to G5/G6) and mark its sub-test `t.Skip` with the reclassification reason —
  never silently dropped.
- [ ] **Step 3: Run** — each case green or explicitly reclassified.

Run: `go test ./kernel/ops/ -run TestFilletRestored1c -v`

- [ ] **Step 4: Commit.**

```bash
git add kernel/ops/fillet_restored_1c_test.go docs/superpowers/specs/2026-07-11-occt-blend-greening-roadmap.md
git commit -m "test(fillet): G1 1c restored-solid triage (fixes + reclassifications)"
```

---

## Phase 4 — Close out

### Task 7: Roadmap update + live check

**Files:**
- Modify: `docs/superpowers/specs/2026-07-11-occt-blend-greening-roadmap.md`

- [ ] **Step 1: Update the roadmap** — mark G1's outcome (P8/V8 green; 1c results; 1a's 6 moved
  to G5 as honest rejections with the apex predicate documented as the G5 entry criterion).
- [ ] **Step 2: Live check** (CLAUDE.md Live tests): drive one partial-cylinder apex case and the
  box-corner through the MCP bridge; confirm the apex case now reports an honest error (no wrong
  render) and the box corner renders correctly. Capture a screenshot.
- [ ] **Step 3: Full scoreboard** — confirm the net movement and no regressions.

Run: `go test ./model/feature/occtparity/ -run TestOCCTBlendScoreboard -v 2>&1 | grep -E 'TOTAL|simple'`

- [ ] **Step 4: Commit.**

```bash
git add docs/superpowers/specs/2026-07-11-occt-blend-greening-roadmap.md
git commit -m "docs(occt-parity): G1 outcome + 1a apex fix criteria handed to G5"
```

## Self-review notes

- **Spec coverage:** 1a guard (Tasks 1-3) ✓; 1b fix (Tasks 4-5) ✓; 1c triage (Task 6) ✓;
  no-silent-wrong floor ✓; zero-false-positive gate ✓; roadmap/live close (Task 7) ✓.
- **Investigation-led branch:** Task 1 decides fix-vs-guard for 1a; the guard is the concrete
  floor, a trivial fix is applied opportunistically if Task 1 surfaces one.
- **Risk:** the `math`/`geom` accessors in Task 2's predicate are named from memory — the
  implementer confirms exact signatures against `math/*.go` and `kernel/geom/*.go` (the session
  verified the field names: `Line{Origin,Dir}`, `Cylinder{Origin,AxisDir}`, `Cone{Apex,AxisDir}`,
  `Sphere{Center}`).
- **Type consistency:** `revolutionAxisApexEdge`, `commonQuadricNeighbour`, `lineIsQuadricAxis`
  used identically across Tasks 2-3.
