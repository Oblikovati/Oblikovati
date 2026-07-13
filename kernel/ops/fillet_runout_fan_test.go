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

// TestClassifyEndCornersV3 gates the detector on the fixture that motivated the feature: V3's picked
// edge terminates at a valence-5 vertex, which must become exactly ONE endCornerFan of 3 planar far
// faces separated by 2 interior far edges, and that vertex must be marked owned so the trihedral ends
// path skips it. Its other end (valence 3) stays trihedral and produces no fan.
func TestClassifyEndCornersV3(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	fils := solvedFilsForCase(t, b, "simple/V3")
	fans, fanV := classifyEndCorners(fils)
	if len(fans) != 1 {
		t.Fatalf("V3: got %d fans, want 1 (the valence-5 end)", len(fans))
	}
	f := fans[0]
	if len(f.fan) != 3 || len(f.farEdges) != 2 {
		// Fatal, not Errorf: assertFanChainOrder below indexes fan/farEdges unconditionally,
		// so a short fan must abort this test cleanly rather than panic the whole binary.
		t.Fatalf("V3 fan: %d far faces / %d far edges, want 3 / 2", len(f.fan), len(f.farEdges))
	}
	if !fanV[vertexNear(t, b, math.P3(34.2, 94, 50)).ID()] {
		t.Error("V3: valence-5 vertex not marked a fan vertex")
	}
	assertFanChainOrder(t, f)
}

// assertFanChainOrder locks the cyclic A-flank -> B-flank walk itself, not just the face/edge
// counts: the sentinels at both flanks, that consecutive fan faces are chained by sharing the
// interior far edge between them, and that each far edge's left/right face matches the fan
// entries on either side of it (in the orientation buildEndCornerFan/farEdgesOf actually produce:
// farEdges[i].leftFace is the face BEFORE the edge in the chain, rightFace the face AFTER).
func assertFanChainOrder(t *testing.T, f endCornerFan) {
	t.Helper()
	if f.fan[0].entryEdge != 0 {
		t.Errorf("V3 fan[0].entryEdge = %d, want 0 (A-flank sentinel)", f.fan[0].entryEdge)
	}
	last := len(f.fan) - 1
	if f.fan[last].exitEdge != 0 {
		t.Errorf("V3 fan[%d].exitEdge = %d, want 0 (B-flank sentinel)", last, f.fan[last].exitEdge)
	}
	for i, fe := range f.farEdges {
		if f.fan[i].exitEdge != fe.edge {
			t.Errorf("V3 fan[%d].exitEdge = %d, want farEdges[%d].edge = %d", i, f.fan[i].exitEdge, i, fe.edge)
		}
		if f.fan[i+1].entryEdge != fe.edge {
			t.Errorf("V3 fan[%d].entryEdge = %d, want farEdges[%d].edge = %d", i+1, f.fan[i+1].entryEdge, i, fe.edge)
		}
		if fe.leftFace != f.fan[i].face {
			t.Errorf("V3 farEdges[%d].leftFace = %d, want fan[%d].face = %d", i, fe.leftFace, i, f.fan[i].face)
		}
		if fe.rightFace != f.fan[i+1].face {
			t.Errorf("V3 farEdges[%d].rightFace = %d, want fan[%d].face = %d", i, fe.rightFace, i+1, f.fan[i+1].face)
		}
	}
}

// solvedFilsForCase solves a wired corpus case's real pick edge (the runout edge the corpus locator
// selects, verified against corpus.json's Locator) exactly as production does. The edge is located
// coordinate-robustly by its two end vertices (vertexNear tolerates the fixture's exact-but-noisy
// coords), NOT by a hard-coded endpoint: the real V3 v5 sits at y=93.969, so edgeByEndpoints' 1e-3
// tol against a rounded (34.2,94,50) would miss it.
func solvedFilsForCase(t *testing.T, b *topo.Body, rel string) []edgeFillet {
	t.Helper()
	pa, pb, r, ok := filletPickForCase(rel)
	if !ok {
		t.Fatalf("solvedFilsForCase: case %q is not wired", rel)
	}
	va := vertexNear(t, b, pa)
	vb := vertexNear(t, b, pb)
	e := edgeBetween(t, b, va, vb)
	fil, err := computeEdgeFillet(b, filletPick{edge: e, r0: r, r1: r},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward)
	if err != nil {
		t.Fatalf("%s computeEdgeFillet: %v", rel, err)
	}
	return []edgeFillet{fil}
}

// filletPickForCase gives the two runout-edge endpoints and pick radius (from corpus.json) for each
// wired corpus case, so solvedFilsForCase resolves the real pick edge for V3 (valence-5 end) and V5
// (valence-6 end) alike. Endpoints are the pick's midpoint ± direction·length/2 from the case's
// Locator, rounded — vertexNear snaps them to the fixture's exact vertices.
func filletPickForCase(rel string) (a, b math.Point3, radius float64, ok bool) {
	switch rel {
	case "simple/V3":
		return math.P3(34.2, 94, 50), math.P3(-0.612, 86, 59.7), 5, true
	case "simple/V5":
		return math.P3(42.26, 90.63, 50), math.P3(-36.25, 16.91, 25.82), 5, true
	case "simple/V1":
		// V1 = OCCT wedge scaled ×10; the pick edge runs apex (50,70,50) to start (0,0,100), r=5.
		return math.P3(50, 70, 50), math.P3(0, 0, 100), 5, true
	}
	return math.Point3{}, math.Point3{}, 0, false
}

// edgeBetween returns the body edge whose two endpoints are exactly the vertices p and q.
func edgeBetween(t *testing.T, b *topo.Body, p, q *topo.Vertex) *topo.Edge {
	t.Helper()
	for _, e := range b.Edges() {
		if s, u := e.StartVertex(), e.EndVertex(); (s == p && u == q) || (s == q && u == p) {
			return e
		}
	}
	t.Fatalf("no edge between %v and %v", p.Point(), q.Point())
	return nil
}

// TestVertexValence pins vertexValence against the occtparity corpus fixtures that motivated this
// feature: V3 has a valence-5 vertex (the runout end that OCCT fillets and this kernel currently
// drops a face at), and V5 has a valence-6 vertex. The exact coordinates come from the fixture's
// own STEP data (see fillet_maxwidth_test.go's use of the same points for V3), so a mismatch here
// means the fixture changed underneath us, not that the geometry moved.
func TestVertexValence(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	if got := vertexValence(vertexNear(t, b, math.P3(34.2, 94, 50))); got != 5 {
		t.Errorf("V3 vertex near (34.2,94,50) valence = %d, want 5", got)
	}
	if got := vertexValence(vertexNear(t, b, math.P3(-0.612, 86, 59.7))); got != 3 {
		t.Errorf("V3 vertex near (-0.612,86,59.7) valence = %d, want 3", got)
	}

	// V5's exact runout-vertex coordinates aren't pinned in this task's brief, so assert the
	// INTENT coordinate-robustly: the fixture must contain at least one valence-6 vertex.
	b5 := importCorpusSolid(t, "simple/V5")
	maxVal := 0
	for _, e := range b5.Edges() {
		for _, v := range []*topo.Vertex{e.StartVertex(), e.EndVertex()} {
			if got := vertexValence(v); got > maxVal {
				maxVal = got
			}
		}
	}
	if maxVal < 6 {
		t.Errorf("V5 max vertex valence = %d, want >= 6", maxVal)
	}
}

// TestClassifyEndCornersExcludesKGreaterThanOne locks the k>=2 boundary (Task 2): a corner where
// TWO filleted edges meet must never become an endCornerFan, even at a >3-valent vertex where the
// fan detector's valence gate alone would not exclude it — it is the corner.miter/blend flags, set
// by computeCorners BEFORE any fils exist, that do the excluding.
//
// The fixture is V3's own valence-5 runout vertex (already proven fan-eligible by valence in
// TestClassifyEndCornersV3), picking two of its OTHER four edges (10 and 24) that share face 25:
// a real miter configuration, not a synthetic body, confirming the boundary on production geometry.
// computeCorners resolves any >=2-filleted-edge vertex to a miter/blend or errors outright before
// computeFillets ever runs — so a corner with c.blend==false && c.miter==false is structurally only
// reachable at k<=1, and fanForEndCorner's existing `c.blend || c.miter` gate already excludes k>=2
// without any code change.
func TestClassifyEndCornersExcludesKGreaterThanOne(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	v := vertexNear(t, b, math.P3(34.202014332567, 93.969262078591, 50))
	if got := vertexValence(v); got <= 3 {
		t.Fatalf("setup: vertex valence = %d, want >3 (fan-eligible by valence alone)", got)
	}
	e1 := edgeBetween(t, b, v, vertexNear(t, b, math.P3(0, 0, 0)))
	e2 := edgeBetween(t, b, v, vertexNear(t, b, math.P3(93.969262078591, -34.20201433256, 0)))
	picks := []filletPick{{edge: e1, r0: 1, r1: 1}, {edge: e2, r0: 1, r1: 1}}

	blends, miters, err := computeCorners(picks)
	if err != nil {
		t.Fatalf("computeCorners: %v", err)
	}
	if miters[v.ID()] == nil {
		t.Fatalf("setup: expected a miter at the shared vertex (blend=%v)", blends[v.ID()] != nil)
	}

	fils := make([]edgeFillet, len(picks))
	for i, p := range picks {
		fil, err := computeEdgeFillet(b, p, blends, miters, FillConcaveOutward)
		if err != nil {
			t.Fatalf("computeEdgeFillet: %v", err)
		}
		fils[i] = fil
	}

	fans, owned := classifyEndCorners(fils)
	if len(fans) != 0 {
		t.Errorf("k=2 miter corner produced %d fans, want 0", len(fans))
	}
	if owned[v.ID()] {
		t.Errorf("k=2 miter vertex marked fan-owned, want untouched (belongs to addCornerRound/miter)")
	}
}

// TestVertexFacesDeduplicates covers the helper vertexValence is built on: a vertex incident to
// several edges that share the same face must count that face once, not once per edge.
func TestVertexFacesDeduplicates(t *testing.T) {
	b := importCorpusSolid(t, "simple/V3")
	v := vertexNear(t, b, math.P3(-0.612, 86, 59.7))
	faces := vertexFaces(v)
	seen := map[uint64]bool{}
	for _, f := range faces {
		if seen[f.ID()] {
			t.Fatalf("vertexFaces returned duplicate face id %d", f.ID())
		}
		seen[f.ID()] = true
	}
	if len(faces) != vertexValence(v) {
		t.Errorf("len(vertexFaces)=%d != vertexValence=%d", len(faces), vertexValence(v))
	}
}
