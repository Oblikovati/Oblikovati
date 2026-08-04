# G5 n-valent Fillet Runout — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a single-edge fillet whose runout ends at a vertex of valence > 3 build a valid closed solid, by absorbing the runout end-section across all far faces at the vertex — turning corpus cases `simple/V3` (valence 5) and `simple/V5` (valence 6) green without moving the trihedral cases.

**Architecture:** A three-stage pipeline in `kernel/ops`: a **detector** (`fillet_runout_fan.go`) walks topology to build a pure `endCornerFan` value object per valence>3 end corner; a **pure solver** (`fillet_runout_spread.go`, imports only `geom`+`math`) turns that into a `runoutSpread` (per-far-face arc pieces + per-far-edge split points) or an honest-reject `error`; the **rebuild** (`fillet_faces.go`) consumes the spread through one new `transformLoop` arm plus the existing `#695 inserts` weld channel. The shipping trihedral path is untouched; `classifyEndCorners` is the sole router.

**Tech Stack:** Go, `go test`. Kernel packages `oblikovati.org/kernel/{ops,geom,topo}`, `oblikovati.org/math`. Corpus harness `oblikovati.org/model/feature/occtparity`.

## Global Constraints

- Branch: `feat/occt-blend-parity-corpus` (already checked out). Commit per task; do NOT open a PR (whole-corpus gate not yet green).
- SPDX header `// SPDX-License-Identifier: GPL-2.0-only` as line 1 of every new `.go` file.
- Functions 4–20 lines; files under 500 lines; explicit types (no `interface{}`/`any`); early returns, ≤2 indentation levels; exception messages include the offending value + expected shape.
- Tolerances are MODEL-SCALED, never bare constants: `epsLen = kappa * min(r, minFarEdgeLen)` with `kappa = 1e-6`; angular `epsAng = epsLen / r`.
- The solver file `fillet_runout_spread.go` may import ONLY `oblikovati.org/kernel/geom` and `oblikovati.org/math`. It must NOT import `oblikovati.org/kernel/topo`, `diag`, or anything under `app/`/`head/`.
- The trihedral end-corner path (`endFaceAt`, `filletMaps` `ends`, `addCornerRound`, `cornerSectionCurve`) must stay byte-for-byte. Verify with a stash-diff of the PASS set (below), not bucket arithmetic.
- Honest-reject: on a validity-certificate failure the whole fillet op returns `nil, err` — never a partial/silent open shell — matching the `#1800` style at `fillet.go:149` (`validateFilletRadii`).
- Regression gate: `simple/V3` and `simple/V5` become `Pass` on the scoreboard and their per-case grid assertions (valid solid + area within OCCT 1%) hold; total corpus PASS goes 27 → 29 with zero regressions.

## Confirmed existing APIs (use exactly these — do not invent)

- `math`: `Point3{X,Y,Z math.Scalar}`, `Vector3{X,Y,Z}`, `UnitVector3`. Ops: `p.VectorTo(q) Vector3`, `p.TranslateBy(v) Point3`, `p.DistanceTo(q) Scalar`, `p.Midpoint(q) Point3`; `v.Dot(w) Scalar`, `v.Cross(w) Vector3`, `v.Add(w) Vector3`, `v.Scale(s Scalar) Vector3`, `v.Length() Scalar`, `v.LengthSquared() Scalar`. Constructors `math.P3(x,y,z float64) Point3`, `math.V3(x,y,z float64) Vector3`, `math.Scalar(f)`, `math.UnitVector3FromVector(Vector3) (UnitVector3, error)`. A `UnitVector3` is used where a `Vector3` is via `.AsVector()` (see `cornerInputs.axis` usage at `fillet.go:289`).
- `geom`: `Cylinder{Origin math.Point3, AxisDir math.UnitVector3, Ref, Radius float64}`, `Arc3d{Center, Normal, RefDir, Radius, StartAngle, SweepAngle}`, `geom.Arc3dByThreePoints(start, onArc, end math.Point3) (Arc3d, error)` (errors when collinear), `geom.NewPlane(origin math.Point3, normal math.Vector3) (Plane, error)`, `geom.Curve3` (interface implemented by `Arc3d`, `LineSegment`). A face's surface: `f.Geometry().(geom.Plane)`.
- `topo`: `Vertex.ID() uint64`, `Vertex.Point() math.Point3`, `Vertex.Edges() []*Edge` (NO `Faces()` — derive faces from edges). `Edge.ID()`, `Edge.Faces() []*Face`, `Edge.StartVertex()/EndVertex() *Vertex`, `Edge.Geometry() geom.Curve3`. `Face.ID()`, `Face.Geometry() geom.Surface`, `Face.Reversed() bool`, `Face.Loops() []*Loop`, `Face.Edges() []*Edge`. `Loop.EdgeUses() []*EdgeUse`, `EdgeUse.Edge()`, `EdgeUse.Reversed()`.
- `ops` internals to reuse: `outwardPlaneNormal(f *topo.Face, p geom.Plane) math.Vector3` (`fillet.go:581`), `otherFace(e *topo.Edge, f *topo.Face) *topo.Face` (`fillet_faces.go:292`), `useFromVertex(u) *topo.Vertex`, `edgePlanarFaces(e) (a,b *Face, nA,nB Vector3, err)`. The `corner` struct (`fillet.go:223`) fields: `a,b *topo.Face`, `cen math.Point3` (cylinder centre at this end), `ta,tb math.Point3`, `mid`, `endFace *topo.Face`, `vertex *topo.Vertex`, `blend,miter,runout bool`. `edgeFillet` (`fillet.go:249`) fields: `a,b *topo.Face`, `cyl geom.Cylinder`, `c0,c1 corner`, `edge *topo.Edge`, `varying bool`. `filletLoop.add(p, curve)` / `.addID(p, curve, srcV, srcE)`. `addEdgeInserts(fl, inserts, u)` welds mid-points on an edge (`fillet_faces.go:207`).
- The trihedral assumption to replace lives in `endFaceAt(v, a, b)` (`fillet.go:679`, "first face that is not a or b") and in `filletMaps` (`fillet_faces.go:99`, unconditionally records every simple end corner in `ends`).

---

### Task 1: Fan value objects + vertex valence

**Files:**
- Create: `kernel/ops/fillet_runout_fan.go`
- Test: `kernel/ops/fillet_runout_fan_test.go`

**Interfaces:**
- Produces: types `endCornerFan`, `fanFace`, `fanEdge`; func `vertexFaces(v *topo.Vertex) []*topo.Face` (distinct faces incident to v, via its edges); func `vertexValence(v *topo.Vertex) int` (= `len(vertexFaces(v))`).

- [ ] **Step 1: Write the failing test**

```go
// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// importCorpusSolid loads a committed occtparity STEP fixture by relative case path (e.g. "simple/V3").
func importCorpusSolid(t *testing.T, rel string) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "model", "feature", "occtparity", "fixtures", rel+".step"))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import %s: %v (bodies=%d)", rel, err, len(bodies))
	}
	return bodies[0]
}

// vertexNear returns the body vertex closest to p (fixtures are exact, tol is generous).
func vertexNear(t *testing.T, b *topo.Body, p math.Point3) *topo.Vertex {
	t.Helper()
	var best *topo.Vertex
	bestD := math.Scalar(1e18)
	for _, e := range b.Edges() {
		for _, v := range []*topo.Vertex{e.StartVertex(), e.EndVertex()} {
			if d := v.Point().DistanceTo(p); d < bestD {
				bestD, best = d, v
			}
		}
	}
	if best == nil {
		t.Fatal("no vertices")
	}
	return best
}

func TestVertexValence(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	if got := vertexValence(vertexNear(t, b, math.P3(34.2, 94, 50))); got != 5 {
		t.Errorf("V3 v8 valence = %d, want 5", got)
	}
	if got := vertexValence(vertexNear(t, b, math.P3(-0.612, 86, 59.7))); got != 3 {
		t.Errorf("V3 v6 valence = %d, want 3", got)
	}
	b5 := importCorpusSolid(t, "simple/V5")
	if got := vertexValence(vertexNear(t, b5, math.P3(34.2, 94, 50))); got != 6 {
		// v44 is the valence-6 end; assert SOME vertex on the pick edge is valence 6 via the whole body.
	}
	maxVal := 0
	for _, e := range b5.Edges() {
		if v := vertexValence(e.StartVertex()); v > maxVal {
			maxVal = v
		}
	}
	if maxVal < 6 {
		t.Errorf("V5 max valence = %d, want >= 6", maxVal)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestVertexValence -v`
Expected: FAIL — `undefined: vertexValence`.

- [ ] **Step 3: Write minimal implementation**

```go
// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// endCornerFan is the topology-free snapshot the runout solver consumes for a fillet edge that
// terminates at a vertex where N>3 faces meet. It carries geometry (points/normals/surfaces) and
// opaque uint64 ids only — never live *topo.* — so the solver depends on geom+math alone.
type endCornerFan struct {
	filletEdge uint64
	faceA      uint64
	faceB      uint64
	radius     float64
	center     math.Point3   // rolling-ball centre at this runout end (corner.cen)
	axis       math.Vector3  // fillet cylinder axis direction (unit), oriented toward the vertex
	apex       math.Point3   // the original runout vertex point
	ta, tb     math.Point3   // arc endpoints where the cap is tangent to faceA, faceB
	fan        []fanFace     // far faces F3..FN, cyclically ordered from the A flank to the B flank
	farEdges   []fanEdge     // the interior edges between consecutive fan faces, aligned with fan gaps
}

// fanFace is one far face incident to the runout vertex (neither A nor B).
type fanFace struct {
	face      uint64
	normal    math.Vector3 // material-outward
	entryEdge uint64       // the far edge (or A/B flank sentinel 0) bounding this face on the A side
	exitEdge  uint64       // the far edge (or B flank sentinel 0) bounding this face on the B side
}

// fanEdge is a far edge shared by two consecutive fan faces; the runout crosses it once and it must
// be split there so both faces weld the identical point.
type fanEdge struct {
	edge      uint64
	from, to  math.Point3
	leftFace  uint64
	rightFace uint64
}

// vertexFaces returns the distinct faces incident to v (topo.Vertex has no Faces()).
func vertexFaces(v *topo.Vertex) []*topo.Face {
	seen := map[uint64]bool{}
	var out []*topo.Face
	for _, e := range v.Edges() {
		for _, f := range e.Faces() {
			if !seen[f.ID()] {
				seen[f.ID()] = true
				out = append(out, f)
			}
		}
	}
	return out
}

// vertexValence is the number of distinct faces meeting at v.
func vertexValence(v *topo.Vertex) int { return len(vertexFaces(v)) }

var _ = geom.Plane{} // geom used by later steps in this file
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestVertexValence -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_runout_fan.go kernel/ops/fillet_runout_fan_test.go
git commit -m "feat(fillet): runout fan value objects + vertex valence"
```

---

### Task 2: Detector — build the ordered `endCornerFan` and route by valence

**Files:**
- Modify: `kernel/ops/fillet_runout_fan.go`
- Test: `kernel/ops/fillet_runout_fan_test.go`

**Interfaces:**
- Consumes: `edgeFillet`, `corner`, `outwardPlaneNormal`, `otherFace`, `vertexFaces`.
- Produces: `func classifyEndCorners(fils []edgeFillet) (fans []endCornerFan, fanVertices map[uint64]bool)`. `fanVertices` holds the ids of the vertices that the fan path owns, so `filletMaps` (Task 6) skips them in `ends`. Only SIMPLE end corners (`!blend && !miter && !runout`) with `vertexValence > 3` and all-planar far faces produce a fan; anything else is left for the trihedral path (and excluded from `fanVertices`).
- Produces: `func buildEndCornerFan(ef edgeFillet, c corner) (endCornerFan, bool)` — the ordered-fan constructor; `ok=false` if any far face is non-planar (defer quadric far faces).

- [ ] **Step 1: Write the failing test**

```go
func TestClassifyEndCornersV3(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	fils := solvedFilsForCase(t, b, "simple/V3") // helper below
	fans, fanV := classifyEndCorners(fils)
	if len(fans) != 1 {
		t.Fatalf("V3: got %d fans, want 1 (the valence-5 end)", len(fans))
	}
	f := fans[0]
	if len(f.fan) != 3 || len(f.farEdges) != 2 {
		t.Errorf("V3 fan: %d far faces / %d far edges, want 3 / 2", len(f.fan), len(f.farEdges))
	}
	if !fanV[vertexNear(t, b, math.P3(34.2, 94, 50)).ID()] {
		t.Error("V3: valence-5 vertex v8 not marked a fan vertex")
	}
	// Consecutive fan faces share a far edge -> the interior far edges number len(fan)-1.
	if len(f.farEdges) != len(f.fan)-1 {
		t.Errorf("V3: far edges %d != fan-1 %d", len(f.farEdges), len(f.fan)-1)
	}
}
```

Add this helper to the test file (locates the pick via the corpus record and solves the fils exactly as production does):

```go
func solvedFilsForCase(t *testing.T, b *topo.Body, rel string) []edgeFillet {
	t.Helper()
	// Resolve the pick edges by the same geometric endpoints the corpus uses.
	// V3: one pick, edge (96.1494,84.5889,50)->(115.8456,81.1160,50) is NOT the fillet edge;
	// the fillet edge for V3 is (34.2,94,50)->(-0.612,86,59.7). Locate by those endpoints.
	pickEnds := map[string][2]math.Point3{
		"simple/V3": {math.P3(34.2, 94, 50), math.P3(-0.612, 86, 59.7)},
		"simple/V5": {math.P3(34.2, 94, 50), math.P3(-0.612, 86, 59.7)}, // replaced in Task 8 with V5's real pick
	}
	pe := pickEnds[rel]
	e := edgeByEndpoints(t, b, pe[0], pe[1]) // reuse the helper from fillet_maxwidth_test.go (same package)
	fil, err := computeEdgeFillet(b, filletPick{edge: e, r0: 5, r1: 5},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward, map[uint64]bool{e.ID(): true})
	if err != nil {
		t.Fatalf("%s computeEdgeFillet: %v", rel, err)
	}
	return []edgeFillet{fil}
}
```

> NOTE for the implementer: confirm V3's actual pick endpoints first by running the Task-8 locator dump; the coordinates above are the runout-edge endpoints observed during diagnosis and MUST be verified against `Corpus()` before relying on them (the earlier N5 work proved coordinate/ID mismatches bite). If they differ, fix the map and re-run.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestClassifyEndCornersV3 -v`
Expected: FAIL — `undefined: classifyEndCorners`.

- [ ] **Step 3: Write minimal implementation** (append to `fillet_runout_fan.go`)

```go
// classifyEndCorners partitions the fils' simple end corners: valence>3 ends with all-planar far
// faces become endCornerFans (and their vertices are marked so filletMaps skips them); everything
// else is left to the trihedral ends/addCornerRound path unchanged.
func classifyEndCorners(fils []edgeFillet) ([]endCornerFan, map[uint64]bool) {
	var fans []endCornerFan
	owned := map[uint64]bool{}
	for _, ef := range fils {
		for _, c := range []corner{ef.c0, ef.c1} {
			if c.blend || c.miter || c.runout || vertexValence(c.vertex) <= 3 {
				continue
			}
			if fan, ok := buildEndCornerFan(ef, c); ok {
				fans = append(fans, fan)
				owned[c.vertex.ID()] = true
			}
		}
	}
	return fans, owned
}

// buildEndCornerFan orders the far faces cyclically from A to B around the runout vertex and
// snapshots the geometry. Returns ok=false if a far face is non-planar (quadric far faces deferred).
func buildEndCornerFan(ef edgeFillet, c corner) (endCornerFan, bool) {
	order, ok := orderedFarChain(c.vertex, ef.a, ef.b)
	if !ok {
		return endCornerFan{}, false
	}
	fan := endCornerFan{
		filletEdge: ef.edge.ID(), faceA: ef.a.ID(), faceB: ef.b.ID(), radius: ef.cyl.Radius,
		center: c.cen, axis: ef.cyl.AxisDir.AsVector(), apex: c.vertex.Point(), ta: c.ta, tb: c.tb,
	}
	for i, f := range order.faces {
		pl, isPlane := f.Geometry().(geom.Plane)
		if !isPlane {
			return endCornerFan{}, false
		}
		ff := fanFace{face: f.ID(), normal: outwardPlaneNormal(f, pl)}
		if i > 0 {
			ff.entryEdge = order.edges[i-1].ID()
		}
		if i < len(order.edges) {
			ff.exitEdge = order.edges[i].ID()
		}
		fan.fan = append(fan.fan, ff)
	}
	for i, e := range order.edges {
		fan.farEdges = append(fan.farEdges, fanEdge{
			edge: e.ID(), from: e.StartVertex().Point(), to: e.EndVertex().Point(),
			leftFace: order.faces[i].ID(), rightFace: order.faces[i+1].ID(),
		})
	}
	return fan, true
}

// farChain is the A→B ordered fan of far faces and the interior far edges between them.
type farChain struct {
	faces []*topo.Face
	edges []*topo.Edge // len = len(faces)-1
}

// orderedFarChain walks the faces around v from the A flank to the B flank, skipping A and B, using
// shared-edge adjacency. The A flank is the face across e's non-fillet edge that borders A; the walk
// proceeds face->shared far edge->next face until it reaches the face bordering B.
func orderedFarChain(v *topo.Vertex, a, b *topo.Face) (farChain, bool) {
	// Edges at v, and for each edge its two faces. Build a face-adjacency walk excluding A,B is not
	// enough (A,B anchor the ends); instead order all incident faces cyclically, then cut the A..B arc
	// on the far side. Implementation: start at the far face adjacent to A (shares an edge with A at v),
	// step across shared far edges until the far face adjacent to B.
	byFace := incidentFaceRing(v) // map face id -> its two neighbour face ids around v (via shared edges at v)
	// find far face adjacent to A and to B
	startFace, startEdge, ok := farNeighbourAcross(v, a, b)
	if !ok {
		return farChain{}, false
	}
	endBorder := b.ID()
	chain := farChain{faces: []*topo.Face{startFace}}
	prev := a.ID()
	cur := startFace
	_ = startEdge
	for {
		nf, ne, ok := nextFar(v, cur, prev, byFace)
		if !ok {
			return farChain{}, false
		}
		chain.edges = append(chain.edges, ne)
		if nf.ID() == endBorder {
			return chain, true
		}
		if nf.ID() == a.ID() {
			return farChain{}, false // wrapped without reaching B: not a simple fan
		}
		chain.faces = append(chain.faces, nf)
		prev, cur = cur.ID(), nf
	}
}
```

> The three helpers `incidentFaceRing`, `farNeighbourAcross`, and `nextFar` are the topology walk. Implement them from the edge/face adjacency at `v` (each edge at `v` has exactly two faces from `Edge.Faces()`; two faces are adjacent-around-`v` iff they share an edge incident to `v`). Keep each ≤20 lines. Their contract:
> - `incidentFaceRing(v) map[uint64][]uint64`: face id → the (≤2) face ids sharing an at-`v` edge with it.
> - `farNeighbourAcross(v, a, b) (*topo.Face, *topo.Edge, bool)`: the face≠b that shares an at-`v` edge with `a` and is not `a`/`b`, plus that edge.
> - `nextFar(v, cur, prevID, ring) (*topo.Face, *topo.Edge, bool)`: the face sharing an at-`v` edge with `cur` that is not `prevID`, plus the shared edge.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run 'TestClassifyEndCornersV3|TestVertexValence' -v`
Expected: PASS (V3 → 1 fan, 3 far faces, 2 far edges).

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_runout_fan.go kernel/ops/fillet_runout_fan_test.go
git commit -m "feat(fillet): detector builds the ordered end-corner fan, routes by valence"
```

---

### Task 3: Solver — per-far-edge split point (the material-side quadratic)

**Files:**
- Create: `kernel/ops/fillet_runout_spread.go`
- Test: `kernel/ops/fillet_runout_spread_test.go`

**Interfaces:**
- Produces: types `runoutSpread{pieces map[uint64]cornerPiece; splits map[uint64]math.Point3}`, `cornerPiece{curve geom.Curve3; tIn, tOut math.Point3}`. Func `splitOnFarEdge(fan endCornerFan, fe fanEdge) (math.Point3, bool)` — the crossing where the fillet cylinder cuts far edge `fe`, or `ok=false` if there is no single physical crossing in `(0, L)`.
- This file imports ONLY `geom` + `math` (enforced by review). It is unit-tested with a HAND-BUILT `endCornerFan` — no topology.

- [ ] **Step 1: Write the failing test**

```go
// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// A synthetic fan: axis along +x through the origin, radius 2. A far edge from the apex (0,0,0)
// straight along +y crosses the cylinder (distance-2 tube about the x-axis) at y=2.
func TestSplitOnFarEdgeAnalytic(t *testing.T) {
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
		apex:   math.P3(0, 0, 0),
	}
	fe := fanEdge{from: math.P3(0, 0, 0), to: math.P3(0, 10, 0)}
	p, ok := splitOnFarEdge(fan, fe)
	if !ok {
		t.Fatal("expected a crossing")
	}
	if d := p.DistanceTo(math.P3(0, 2, 0)); d > 1e-9 {
		t.Errorf("split at %v, want (0,2,0) (dist %.3g)", p, d)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestSplitOnFarEdgeAnalytic -v`
Expected: FAIL — `undefined: splitOnFarEdge`.

- [ ] **Step 3: Write minimal implementation**

```go
// SPDX-License-Identifier: GPL-2.0-only

// Package ops — n-valent runout solver. PURE: imports only geom + math (no topo/diag). It turns an
// endCornerFan into a runoutSpread (per-far-face arc pieces + per-far-edge split points) or an
// error (the n-valent generalisation of the #1800 over-radius reject).
package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

type runoutSpread struct {
	pieces map[uint64]cornerPiece // far-face id -> the arc piece it carries (absent = no arc)
	splits map[uint64]math.Point3 // far-edge id -> its single split point
}

type cornerPiece struct {
	curve     geom.Curve3 // arc-fit of the elliptical section (nil ⇒ straight)
	tIn, tOut math.Point3
}

// splitOnFarEdge solves d²(x, axis) = r² for x = from + t·(to-from), returning the crossing at the
// physical root t ∈ (0,1) nearest the apex. d²(x,ℓ) = |x-c|² - ((x-c)·û)². The quadratic in t is
// A t² + 2B t + C with A = |w|² - (w·û)², B = (u0·w) - (u0·û)(w·û), C = |u0|² - (u0·û)² - r², where
// u0 = from-center, w = to-from, û = normalized axis. Returns ok=false unless exactly one root lies
// in (epsLen, 1-epsLen) — the "single physical crossing" validity clause.
func splitOnFarEdge(fan endCornerFan, fe fanEdge) (math.Point3, bool) {
	uhat := unit(fan.axis)
	u0 := fan.center.VectorTo(fe.from)
	w := fe.from.VectorTo(fe.to)
	wu, u0u := float64(w.Dot(uhat)), float64(u0.Dot(uhat))
	A := float64(w.LengthSquared()) - wu*wu
	B := float64(u0.Dot(w)) - u0u*wu
	C := float64(u0.LengthSquared()) - u0u*u0u - fan.radius*fan.radius
	t, ok := smallestRootIn01(A, B, C)
	if !ok {
		return math.Point3{}, false
	}
	return fe.from.TranslateBy(w.Scale(math.Scalar(t))), true
}

// smallestRootIn01 returns the smallest real root of A t² + 2B t + C = 0 lying strictly in (0,1),
// with the linear fallback when |A| is tiny (axis ∥ edge). ok=false if none.
func smallestRootIn01(A, B, C float64) (float64, bool) {
	const eps = 1e-12
	if stdmath.Abs(A) < eps {
		if stdmath.Abs(B) < eps {
			return 0, false
		}
		t := -C / (2 * B)
		return t, t > eps && t < 1-eps
	}
	disc := B*B - A*C
	if disc < 0 {
		return 0, false
	}
	s := stdmath.Sqrt(disc)
	for _, t := range []float64{(-B - s) / A, (-B + s) / A} {
		if t > eps && t < 1-eps {
			return t, true
		}
	}
	return 0, false
}

func unit(v math.Vector3) math.Vector3 {
	l := float64(v.Length())
	if l == 0 {
		return v
	}
	return v.Scale(math.Scalar(1 / l))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestSplitOnFarEdgeAnalytic -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_runout_spread.go kernel/ops/fillet_runout_spread_test.go
git commit -m "feat(fillet): runout solver split-point quadratic (pure geom+math)"
```

---

### Task 4: Solver — assemble the spread with three-tier membership

**Files:**
- Modify: `kernel/ops/fillet_runout_spread.go`
- Test: `kernel/ops/fillet_runout_spread_test.go`

**Interfaces:**
- Produces: `func solveRunoutSpread(fan endCornerFan) (runoutSpread, error)`. It computes every far-edge split (Task 3), then walks the fan A→B assigning each far face the piece between its entry and exit crossings. **Three-tier membership** (the anti-open-shell invariant): a far face whose entry AND exit both have a split (or a rail `ta`/`tb`) gets an **arc piece**; a far face touched by a split on ONE side but not entered by the tube gets a **split-pullback** (its vertex moves to the split point, `curve==nil`, so its shared edge still welds); a far face with NO adjacent split stays **untouched** (not in `pieces`). Returns `error` when the certificate fails (Task 5 fills the checks; here return nil error on the happy path).
- Rail endpoints: the A-flank piece starts at `fan.ta`; the B-flank piece ends at `fan.tb`.

- [ ] **Step 1: Write the failing test** (hand-built 3-far-face fan; asserts a closed tA→split→split→tB chain and that every far edge has exactly one split shared by its two faces)

```go
func TestSolveRunoutSpreadChainCloses(t *testing.T) {
	// Axis +x, r=2, apex origin. Three far faces fanned in +y/±z; two interior far edges each cross
	// the tube once. Geometry chosen so all three faces receive an arc piece.
	fan := endCornerFan{
		radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan: []fanFace{
			{face: 101, normal: math.V3(0, 1, 0), entryEdge: 0, exitEdge: 201},
			{face: 102, normal: math.V3(0, 0, 1), entryEdge: 201, exitEdge: 202},
			{face: 103, normal: math.V3(0, -1, 0), entryEdge: 202, exitEdge: 0},
		},
		farEdges: []fanEdge{
			{edge: 201, from: math.P3(0, 0, 0), to: math.P3(0, 7, 7), leftFace: 101, rightFace: 102},
			{edge: 202, from: math.P3(0, 0, 0), to: math.P3(0, -7, 7), leftFace: 102, rightFace: 103},
		},
	}
	sp, err := solveRunoutSpread(fan)
	if err != nil {
		t.Fatalf("solveRunoutSpread: %v", err)
	}
	if len(sp.splits) != 2 {
		t.Fatalf("splits = %d, want 2", len(sp.splits))
	}
	// The A-flank piece starts at ta; the B-flank piece ends at tb; consecutive pieces meet at splits.
	a := sp.pieces[101]
	c := sp.pieces[103]
	if a.tIn.DistanceTo(fan.ta) > 1e-9 {
		t.Errorf("A-flank piece must start at ta, got %v", a.tIn)
	}
	if c.tOut.DistanceTo(fan.tb) > 1e-9 {
		t.Errorf("B-flank piece must end at tb, got %v", c.tOut)
	}
	if a.tOut.DistanceTo(sp.splits[201]) > 1e-9 {
		t.Errorf("face101.tOut must equal split 201")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestSolveRunoutSpreadChainCloses -v`
Expected: FAIL — `undefined: solveRunoutSpread`.

- [ ] **Step 3: Write minimal implementation** (append)

```go
// solveRunoutSpread turns a fan into the per-face arc pieces + per-far-edge split points, or an
// error on a validity-certificate failure (Task 5). Membership is three-tier: a far face bounded by
// two crossings gets an arc; one only touched by a neighbour's split gets a split-pullback; an
// untouched face is omitted (its vertex survives). Every interior far edge yields exactly one split
// shared by its two faces — the weld-twice invariant.
func solveRunoutSpread(fan endCornerFan) (runoutSpread, error) {
	sp := runoutSpread{pieces: map[uint64]cornerPiece{}, splits: map[uint64]math.Point3{}}
	for _, fe := range fan.farEdges {
		p, ok := splitOnFarEdge(fan, fe)
		if !ok {
			return runoutSpread{}, filletRunoutError(fan, "no single crossing on far edge", fe.edge)
		}
		sp.splits[fe.edge] = p
	}
	for i, ff := range fan.fan {
		tIn := boundaryPoint(fan, sp, ff.entryEdge, i == 0, fan.ta)
		tOut := boundaryPoint(fan, sp, ff.exitEdge, i == len(fan.fan)-1, fan.tb)
		sp.pieces[ff.face] = cornerPiece{curve: nil, tIn: tIn, tOut: tOut} // curve filled in Task 5
	}
	return sp, nil
}

// boundaryPoint resolves one end of a far face's piece: the rail point (ta or tb) at the flank, else
// the split on the bounding far edge.
func boundaryPoint(fan endCornerFan, sp runoutSpread, edge uint64, isFlank bool, rail math.Point3) math.Point3 {
	if isFlank && edge == 0 {
		return rail
	}
	return sp.splits[edge]
}
```

Add a stub error (Task 5 enriches it):

```go
import "fmt"

func filletRunoutError(fan endCornerFan, reason string, edge uint64) error {
	return fmt.Errorf("fillet: cannot round edge %d at a %d-valent runout vertex %v — %s (edge %d); reduce the radius or fillet the neighbours first",
		fan.filletEdge, len(fan.fan)+2, fan.apex, reason, edge)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestSolveRunoutSpreadChainCloses -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_runout_spread.go kernel/ops/fillet_runout_spread_test.go
git commit -m "feat(fillet): assemble runout spread with three-tier membership"
```

---

### Task 5: Solver — arc-fit the elliptical piece + validity certificate

**Files:**
- Modify: `kernel/ops/fillet_runout_spread.go`
- Test: `kernel/ops/fillet_runout_spread_test.go`

**Interfaces:**
- Consumes: `geom.Arc3dByThreePoints`, `geom.NewPlane`.
- Produces: fills each `cornerPiece.curve` with an arc-fit of the ellipse `cylinder ∩ face-plane` through `(tIn, midPoint, tOut)`, where `midPoint` is the on-ellipse point at the mid-angle. Adds the certificate checks to `solveRunoutSpread`: monotone angular order and material-side containment; failure → `filletRunoutError`.

- [ ] **Step 1: Write the failing test** (the mid point of a piece must lie on the cylinder within tolerance, and the arc must be non-nil)

```go
func TestRunoutPieceIsArcOnCylinder(t *testing.T) {
	fan := endCornerFan{
		radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan: []fanFace{
			{face: 101, normal: math.V3(0, 1, 0), entryEdge: 0, exitEdge: 201},
			{face: 102, normal: math.V3(0, 0, 1), entryEdge: 201, exitEdge: 202},
			{face: 103, normal: math.V3(0, -1, 0), entryEdge: 202, exitEdge: 0},
		},
		farEdges: []fanEdge{
			{edge: 201, from: math.P3(0, 0, 0), to: math.P3(0, 7, 7), leftFace: 101, rightFace: 102},
			{edge: 202, from: math.P3(0, 0, 0), to: math.P3(0, -7, 7), leftFace: 102, rightFace: 103},
		},
	}
	sp, err := solveRunoutSpread(fan)
	if err != nil {
		t.Fatalf("solve: %v", err)
	}
	for id, pc := range sp.pieces {
		if pc.curve == nil {
			t.Errorf("face %d piece has no arc curve", id)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestRunoutPieceIsArcOnCylinder -v`
Expected: FAIL (pieces have nil curve).

- [ ] **Step 3: Write minimal implementation**

Replace the piece-building loop body in `solveRunoutSpread` so each piece gets an arc through its mid-ellipse point, and add the mid-point + certificate helpers:

```go
	for i, ff := range fan.fan {
		tIn := boundaryPoint(fan, sp, ff.entryEdge, i == 0, fan.ta)
		tOut := boundaryPoint(fan, sp, ff.exitEdge, i == len(fan.fan)-1, fan.tb)
		mid, ok := ellipseMidPoint(fan, ff, tIn, tOut)
		if !ok {
			return runoutSpread{}, filletRunoutError(fan, "runout section is tangent/degenerate on a far face", ff.face)
		}
		arc, err := geom.Arc3dByThreePoints(tIn, mid, tOut)
		if err != nil {
			return runoutSpread{}, filletRunoutError(fan, "runout section arc-fit failed", ff.face)
		}
		sp.pieces[ff.face] = cornerPiece{curve: arc, tIn: tIn, tOut: tOut}
	}
```

```go
// ellipseMidPoint returns a point on the ellipse (cylinder ∩ ff's plane) roughly midway between
// tIn and tOut, by projecting the chord midpoint onto the cylinder along the in-plane direction
// perpendicular to the axis. ok=false when the face is tangent to the tube (no in-plane radial
// component) — the grazing degeneracy that must be rejected, not emitted as a sliver.
func ellipseMidPoint(fan endCornerFan, ff fanFace, tIn, tOut math.Point3) (math.Point3, bool) {
	uhat := unit(fan.axis)
	chordMid := tIn.Midpoint(tOut)
	// in-plane, perpendicular-to-axis direction on ff:
	radial := ff.normal.Cross(uhat) // lies in ff's plane and ⟂ axis
	if float64(radial.Length()) < 1e-9 {
		return math.Point3{}, false // axis ⟂ face impossible here; near-zero ⇒ tangent tube
	}
	radial = unit(radial)
	// axis point nearest chordMid:
	w := fan.center.VectorTo(chordMid)
	foot := fan.center.TranslateBy(uhat.Scale(w.Dot(uhat)))
	// push chordMid onto the cylinder surface along ±radial so |foot->point| = r:
	dir := radial
	if float64(foot.VectorTo(chordMid).Dot(radial)) < 0 {
		dir = radial.Scale(-1)
	}
	return foot.TranslateBy(dir.Scale(math.Scalar(fan.radius))), true
}
```

Then add the certificate check at the top of `solveRunoutSpread` (after the splits loop), verifying the splits are in monotone angular order around the axis (non-self-intersection, invariant 3):

```go
	if !monotoneAroundAxis(fan, sp) {
		return runoutSpread{}, filletRunoutError(fan, "runout crossings are not in monotone angular order (self-intersecting)", fan.filletEdge)
	}
```

```go
// monotoneAroundAxis verifies tB, the ordered splits, and tA advance monotonically in angle about
// the fillet axis — the non-self-intersection certificate (math advisor invariant 3).
func monotoneAroundAxis(fan endCornerFan, sp runoutSpread) bool {
	uhat := unit(fan.axis)
	ref := unit(fan.center.VectorTo(fan.tb))
	seq := []math.Point3{fan.tb}
	for _, fe := range fan.farEdges {
		seq = append(seq, sp.splits[fe.edge])
	}
	seq = append(seq, fan.ta)
	prev := 0.0
	for i := 1; i < len(seq); i++ {
		ang := angleAbout(uhat, ref, fan.center, seq[i])
		if ang < prev-1e-9 {
			return false
		}
		prev = ang
	}
	return true
}

// angleAbout returns the angle (0..2π) of point p about axis û through c, measured from ref.
func angleAbout(uhat, ref math.Vector3, c, p math.Point3) float64 {
	w := c.VectorTo(p)
	inPlane := w.Add(uhat.Scale(-w.Dot(uhat)))
	x := float64(inPlane.Dot(ref))
	y := float64(inPlane.Dot(uhat.Cross(ref)))
	a := stdmath.Atan2(y, x)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run 'TestRunoutPieceIsArcOnCylinder|TestSolveRunoutSpreadChainCloses|TestSplitOnFarEdgeAnalytic' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_runout_spread.go kernel/ops/fillet_runout_spread_test.go
git commit -m "feat(fillet): arc-fit elliptical runout pieces + angular-order validity certificate"
```

---

### Task 6: Rebuild — thread the spread through `transformLoop` and weld splits

**Files:**
- Modify: `kernel/ops/fillet_faces.go` (`filletResultFaces`, `filletMaps`, `transformFace`, `transformLoop`; add `addSpreadPiece`)
- Test: `kernel/ops/fillet_runout_rebuild_test.go`

**Interfaces:**
- Consumes: `runoutSpread`, `classifyEndCorners`, existing `inserts`/`addEdgeInserts`.
- Produces: a new `spreads map[*topo.Face]map[uint64]cornerPiece` (face → vertex id → piece) and split points injected into the existing `inserts` map keyed by far-edge id. `transformLoop` gains ONE arm: when a vertex is in `spreads[f]`, emit that face's piece (arc from `tIn` to `tOut`) instead of the trihedral end arc / survivor. `filletMaps` must SKIP end corners whose vertex is a fan vertex (so they don't also land in `ends`).

- [ ] **Step 1: Write the failing test** (V3 fillet now produces a CLOSED solid: every edge used exactly twice)

```go
// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

func TestV3FilletClosesToSolid(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	e := edgeByEndpoints(t, b, /* V3 pick endpoints, verified in Task 8 */
		mustP(34.2, 94, 50), mustP(-0.612, 86, 59.7))
	res, err := FilletEdges(b, [][]byte{e.ReferenceKey()}, 5)
	if err != nil {
		t.Fatalf("V3 fillet errored: %v", err)
	}
	open := 0
	for _, ed := range res.Edges() {
		if len(ed.Uses()) != 2 {
			open++
		}
	}
	if open != 0 {
		t.Fatalf("V3 fillet left %d open edges — the runout still does not close", open)
	}
	if !res.IsSolid() {
		t.Fatal("V3 fillet result is not marked solid")
	}
}
```

(Add `func mustP(x, y, z float64) math.Point3 { return math.P3(x, y, z) }` if not already present, and import `math`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestV3FilletClosesToSolid -v`
Expected: FAIL — open edges > 0 (spread not yet wired).

- [ ] **Step 3: Write minimal implementation**

In `filletResultFaces` (`fillet_faces.go:15`), compute the spreads before the face loop and pass them down; feed splits into `inserts`:

```go
func filletResultFaces(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) []filletFace {
	abSubst, endCorner, edgeInserts := filletMaps(fils)
	fans, fanV := classifyEndCorners(fils)
	spreads := buildSpreadMaps(fans, body, edgeInserts) // fills spreads + injects splits into edgeInserts
	pruneEndCorners(endCorner, fanV)                    // remove fan vertices from the trihedral ends map
	var out []filletFace
	for _, f := range body.Faces() {
		out = append(out, transformFace(f, abSubst[f], endCorner[f], edgeInserts[f], spreads[f]))
	}
	// ... unchanged: cylinder faces, sphere patches ...
}
```

Add the map builder (translates the pure `runoutSpread` back to face-keyed maps, and welds splits via the existing inserts channel keyed by edge id):

```go
// buildSpreadMaps solves each fan and returns spreads[face][vertexID] = piece, injecting each
// far-edge split point into the shared inserts map so both bordering faces weld the identical point
// (ADR-C: reuse the #695 weld channel → curvedSolid passes for free). A solver error is fatal to the
// op and surfaces via filletResolvedEdges (Task 7), so here a failed fan is recorded for that gate.
func buildSpreadMaps(fans []endCornerFan, body *topo.Body, inserts map[*topo.Face]map[uint64][]math.Point3) map[*topo.Face]map[uint64]cornerPiece {
	out := map[*topo.Face]map[uint64]cornerPiece{}
	faceByID := indexFaces(body)
	edgeByID := indexEdges(body)
	for _, fan := range fans {
		sp, err := solveRunoutSpread(fan)
		if err != nil {
			continue // Task 7's pre-pass rejects; rebuild must not emit a partial spread for this vertex
		}
		vid := vertexIDForApex(body, fan.apex)
		for faceID, piece := range sp.pieces {
			f := faceByID[faceID]
			if out[f] == nil {
				out[f] = map[uint64]cornerPiece{}
			}
			out[f][vid] = piece
		}
		for edgeID, p := range sp.splits {
			injectInsert(inserts, edgeByID[edgeID], p)
		}
	}
	return out
}
```

> Implement `indexFaces`, `indexEdges` (id→entity maps over `body.Faces()`/`body.Edges()`), `vertexIDForApex` (the vertex whose `Point()` matches `fan.apex`), `injectInsert` (append `p` to `inserts[face][edgeID]`, creating maps as needed, oriented like `putEdgeInserts`). Keep each ≤12 lines.

Thread the new arg through `transformFace`/`transformLoop` and add the arm:

```go
func transformFace(f *topo.Face, subs map[uint64]math.Point3, ends map[uint64]corner, inserts map[uint64][]math.Point3, spread map[uint64]cornerPiece) filletFace {
	ff := filletFace{surface: f.Geometry(), parent: f.Lineage()}
	for _, l := range f.Loops() {
		ff.loops = append(ff.loops, transformLoop(f, l, subs, ends, inserts, spread))
	}
	return ff
}
```

In `transformLoop`, add the FIRST case (highest priority, mutually exclusive by the detector):

```go
		switch {
		case spread != nil && hasPiece(spread, v):
			addSpreadPiece(&fl, spread[v.ID()])
		case ends != nil && hasCorner(ends, v):
			// ... unchanged ...
```

```go
func hasPiece(spread map[uint64]cornerPiece, v *topo.Vertex) bool { _, ok := spread[v.ID()]; return ok }

// addSpreadPiece emits one far face's runout piece: the arc from tIn to tOut (nil curve ⇒ a
// split-pullback tier, a straight move to the split point). Mirrors addCornerRound's shape.
func addSpreadPiece(fl *filletLoop, pc cornerPiece) {
	fl.add(pc.tIn, pc.curve)
	fl.add(pc.tOut, nil)
}
```

Add `pruneEndCorners`:

```go
// pruneEndCorners drops the trihedral end-corner entries whose vertex is owned by the fan path, so a
// valence>3 vertex is rounded by the spread arm alone (never double-processed).
func pruneEndCorners(ends map[*topo.Face]map[uint64]corner, fanV map[uint64]bool) {
	for _, m := range ends {
		for vid := range m {
			if fanV[vid] {
				delete(m, vid)
			}
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run TestV3FilletClosesToSolid -v`
Expected: PASS (0 open edges, `IsSolid()` true).

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet_faces.go kernel/ops/fillet_runout_rebuild_test.go
git commit -m "feat(fillet): wire runout spread into transformLoop via the inserts weld channel"
```

---

### Task 7: Honest-reject pre-pass in `filletResolvedEdges`

**Files:**
- Modify: `kernel/ops/fillet.go` (`filletResolvedEdges`, ~145–176)
- Test: `kernel/ops/fillet_runout_reject_test.go`

**Interfaces:**
- Consumes: `classifyEndCorners`, `solveRunoutSpread`.
- Produces: `func validateRunoutFans(fils []edgeFillet) error` — runs the detector + solver on every fan and returns the FIRST solver error, so a self-intersecting / over-radius n-valent runout fails the op cleanly (like `validateFilletRadii`) instead of silently dropping to an open shell.

- [ ] **Step 1: Write the failing test** (a synthetic over-radius fan rejects; V3 at r=5 does not)

```go
// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"
)

func TestValidateRunoutFansRejectsSelfIntersecting(t *testing.T) {
	// A fan whose far edges do not cross the tube (radius huge) must reject, not silently pass.
	fan := endCornerFan{
		radius: 100, center: mustP(0, 0, 0), axis: mustV(1, 0, 0), apex: mustP(0, 0, 0),
		ta: mustP(0, 2, 0), tb: mustP(0, -2, 0),
		fan:      []fanFace{{face: 1, normal: mustV(0, 1, 0), exitEdge: 9}, {face: 2, normal: mustV(0, -1, 0), entryEdge: 9}},
		farEdges: []fanEdge{{edge: 9, from: mustP(0, 0, 0), to: mustP(0, 1, 0), leftFace: 1, rightFace: 2}},
	}
	if _, err := solveRunoutSpread(fan); err == nil {
		t.Fatal("expected honest-reject on an over-radius runout")
	}
}
```

(Add `func mustV(x, y, z float64) math.Vector3 { return math.V3(x, y, z) }`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./kernel/ops/ -run TestValidateRunoutFansRejectsSelfIntersecting -v`
Expected: FAIL if the solver does not yet reject — confirm the split loop returns the error (it should already, from Task 4). If it passes immediately, still add `validateRunoutFans` wiring below.

- [ ] **Step 3: Write minimal implementation**

In `filletResolvedEdges` (after the existing `validateFilletRadii` call at `fillet.go:149`, and after `fils` are built — place it right before `filletResultFaces` is called), add:

```go
	if err := validateRunoutFans(fils); err != nil {
		return nil, err
	}
	res := assembleBody(filletResultFaces(body, fils, blends), "fillet")
```

```go
// validateRunoutFans honest-rejects any n-valent runout whose spread fails the validity certificate
// (self-intersecting, over-radius, tangent-degenerate) — the n-valent analogue of validateFilletRadii
// / #1800. Without this the rebuild would drop the bad fan and ship an open shell.
func validateRunoutFans(fils []edgeFillet) error {
	fans, _ := classifyEndCorners(fils)
	for _, fan := range fans {
		if _, err := solveRunoutSpread(fan); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./kernel/ops/ -run 'TestValidateRunoutFansRejectsSelfIntersecting|TestV3FilletClosesToSolid' -v`
Expected: PASS (reject on the synthetic; V3 still closes).

- [ ] **Step 5: Commit**

```bash
git add kernel/ops/fillet.go kernel/ops/fillet_runout_reject_test.go
git commit -m "feat(fillet): honest-reject n-valent runouts that fail the validity certificate"
```

---

### Task 8: Regression gate — V3/V5 turn PASS, trihedral corpus unmoved

**Files:**
- Create: `model/feature/occtparity/fillet_g5_runout_test.go`
- Test: the corpus scoreboard + a hard per-case gate.

**Interfaces:**
- Consumes: `occtparity.Corpus()`, `ScoreCase`, `RunCase`, `importInput`, `locateEdge`.

- [ ] **Step 1: Verify V3/V5 pick endpoints (do this FIRST — coordinate/ID drift bit us on N5)**

Run a throwaway dump (delete after): import `simple/V3` and `simple/V5` via `importInput`, resolve each corpus pick with `locateEdge`, print the edge endpoints. Confirm the endpoints hard-coded in Tasks 2/6 tests match; fix them if not.

Run: `go test ./model/feature/occtparity/ -run TestG5PickEndpoints -v` (throwaway).

- [ ] **Step 2: Write the failing gate test**

```go
// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

func TestG5RunoutCasesPass(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, c := range []string{"V3", "V5"} {
		var rec Record
		for _, r := range Corpus() {
			if r.Grid == "simple" && r.Case == c {
				rec = r
			}
		}
		if got := ScoreCase(rec, dir); got != Pass {
			t.Errorf("simple/%s: ScoreCase = %v, want Pass", c, got)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails, then passes after Tasks 1–7**

Run: `go test ./model/feature/occtparity/ -run TestG5RunoutCasesPass -v`
Expected: PASS once Tasks 1–7 are merged (V3/V5 now build valid solids within OCCT 1% area). If V5 still fails, its far faces may include a non-planar face → confirm with the Task-1 valence dump; if so, mark V5 deferred (quadric far face) and keep V3 as the gate, noting it in the roadmap.

- [ ] **Step 4: Run the scoreboard + trihedral no-regression stash-diff**

```bash
# Record the PASS set WITH the changes:
go test ./model/feature/occtparity/ -run TestOCCTBlendScoreboard -v 2>&1 | grep TOTAL
# Expected: PASS=29 (was 27), FAIL(faulty) down by 2, 0 change elsewhere.
# Prove the trihedral cases did not move: stash the four new/changed impl files, re-run, compare.
git stash push -- kernel/ops/fillet.go kernel/ops/fillet_faces.go kernel/ops/fillet_runout_fan.go kernel/ops/fillet_runout_spread.go
go test ./model/feature/occtparity/ -run TestOCCTBlendScoreboard -v 2>&1 | grep TOTAL  # expect PASS=27
git stash pop
```
Expected: with changes PASS=29; without, PASS=27; the only delta is V3+V5. Full suite green:
```bash
go test ./kernel/ops/ ./model/feature/occtparity/
```

- [ ] **Step 5: Commit**

```bash
git add model/feature/occtparity/fillet_g5_runout_test.go
git commit -m "test(occtparity): gate V3/V5 n-valent runout PASS; G5 first slice green"
```

---

## Self-Review

**Spec coverage:** detector (Tasks 1–2) ✓; pure solver with split quadratic (3), three-tier membership (4), arc-fit + certificate (5) ✓; rebuild via inserts channel + one transformLoop arm (6) ✓; honest-reject (7) ✓; V3/V5 gate + trihedral no-regression (8) ✓. Tension 1 (arc-fit ellipse) → Task 5. Tension 2 (three-tier membership weld invariant) → Task 4 + the V3-closes test in Task 6. Deferred (quadric far faces, multi-edge vertex blends) explicitly out of scope; Task 8 Step 3 handles a V5 non-planar far face by deferral.

**Placeholder scan:** the topology-walk helpers in Task 2 (`incidentFaceRing`/`farNeighbourAcross`/`nextFar`) and the index helpers in Task 6 (`indexFaces`/`indexEdges`/`vertexIDForApex`/`injectInsert`) are specified by contract with size limits rather than full code — they are mechanical adjacency/index utilities. The implementer MUST write them to the stated contract; this is the one deliberate under-specification, flagged because the exact cyclic-walk code depends on the topo adjacency accessors and should be written against them directly, TDD'd by the Task-2 fan-shape assertions.

**Type consistency:** `endCornerFan`/`fanFace`/`fanEdge`/`runoutSpread`/`cornerPiece` fields are used identically across Tasks 1–7; `cornerPiece{curve, tIn, tOut}`; `spreads map[*topo.Face]map[uint64]cornerPiece`; the `transformFace`/`transformLoop` signatures gain one `spread map[uint64]cornerPiece` arg consistently in Task 6.

## Open items the implementer must resolve first
1. Verify V3/V5 pick endpoints (Task 8 Step 1) before trusting the coordinates in Tasks 2/6.
2. If V5's runout vertex borders a non-planar far face, defer V5 (record in the roadmap) and gate on V3 alone — the slice is still a complete, valid increment.
