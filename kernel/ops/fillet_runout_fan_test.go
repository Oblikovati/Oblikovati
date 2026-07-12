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
