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
		t.Errorf("V3 fan: %d far faces / %d far edges, want 3 / 2", len(f.fan), len(f.farEdges))
	}
	if !fanV[vertexNear(t, b, math.P3(34.2, 94, 50)).ID()] {
		t.Error("V3: valence-5 vertex not marked a fan vertex")
	}
	// Consecutive fan faces share a far edge -> the interior far edges number len(fan)-1.
	if len(f.farEdges) != len(f.fan)-1 {
		t.Errorf("V3: far edges %d != fan-1 %d", len(f.farEdges), len(f.fan)-1)
	}
}

// solvedFilsForCase solves V3's real pick edge (the valence-5 -> valence-3 edge the corpus locator
// selects, verified against corpus.json's Locator) exactly as production does. The edge is located
// coordinate-robustly by its two end vertices (vertexNear tolerates the fixture's exact-but-noisy
// coords), NOT by a hard-coded endpoint: the real v5 sits at y=93.969, so edgeByEndpoints' 1e-3 tol
// against a rounded (34.2,94,50) would miss it.
func solvedFilsForCase(t *testing.T, b *topo.Body, rel string) []edgeFillet {
	t.Helper()
	if rel != "simple/V3" {
		t.Fatalf("solvedFilsForCase: only simple/V3 is wired, got %q", rel)
	}
	v5 := vertexNear(t, b, math.P3(34.2, 94, 50))
	v3 := vertexNear(t, b, math.P3(-0.612, 86, 59.7))
	e := edgeBetween(t, b, v5, v3)
	fil, err := computeEdgeFillet(b, filletPick{edge: e, r0: 5, r1: 5},
		map[uint64]*cornerBlend{}, map[uint64]*cornerMiter{}, FillConcaveOutward, map[uint64]bool{e.ID(): true})
	if err != nil {
		t.Fatalf("%s computeEdgeFillet: %v", rel, err)
	}
	return []edgeFillet{fil}
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
