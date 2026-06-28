// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// issuePoly1536 is the irregular quad from bug #1536/#1537 (one corner dragged out).
var issuePoly1536 = []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4.89099465009204, Y: 5.1416568653588115}, {X: 0, Y: 3}}

// keyString1536 renders an edge reference key as a readable string (the \x02 kind byte stripped).
func keyString1536(k []byte) string {
	if len(k) > 0 && k[0] < 0x20 {
		return string(k[1:])
	}
	return string(k)
}

// edgeByKeySuffix1536 returns the reference key of the body edge whose readable key equals want.
func edgeByKeySuffix1536(t *testing.T, b *topo.Body, want string) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if keyString1536(e.ReferenceKey()) == want {
			return e.ReferenceKey()
		}
	}
	t.Fatalf("no edge with key %q", want)
	return nil
}

// verticalEdgeNearFoot1536 returns the straight VERTICAL edge whose foot is closest in XY to (x,y)
// — how the user picks an edge in the viewport, independent of its key.
func verticalEdgeNearFoot1536(t *testing.T, b *topo.Body, x, y float64) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range b.Edges() {
		if _, line := e.Geometry().(geom.LineSegment); !line {
			continue
		}
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(float64(a.X-c.X)) > 1e-6 || stdmath.Abs(float64(a.Y-c.Y)) > 1e-6 {
			continue // not vertical
		}
		if d := stdmath.Hypot(float64(a.X)-x, float64(a.Y)-y); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil {
		t.Fatalf("no vertical edge near (%g,%g)", x, y)
	}
	return best
}

// firstFilletedPrism1536 builds the issue prism, fillets the vertical edge at corner 2
// (Extrusion1:side-edge#2), recomputes, and returns the engine and the filleted body.
func firstFilletedPrism1536(t *testing.T) (*PartFeatures, *topo.Body) {
	t.Helper()
	box := buildPrism(issuePoly1536, sketch.XYPlane(), span{near: 0, far: 5}, 0, "Extrusion1")
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(box)
	edge1 := edgeByKeySuffix1536(t, box, "Extrusion1:side-edge#2")
	f1 := NewDressUpFeatures(fs).AddFillet([][]byte{edge1}, func() float64 { return 1 })
	fs.Recompute()
	if !f1.Health().OK() {
		t.Fatalf("first fillet sick: %+v", f1.Health())
	}
	return fs, fs.Result()[0]
}

// TestSecondFilletWorksWithProvenanceNaming1536 is the #1536/#1537 workflow under ADR-0043 P1: a
// first fillet, then a second fillet on the OPPOSITE vertical edge picked from the displayed body
// (whose key is now a PROVENANCE name, not fillet:e#N). The second fillet must produce real
// geometry — a second cylinder on a valid solid — without disturbing the first.
func TestSecondFilletWorksWithProvenanceNaming1536(t *testing.T) {
	fs, res1 := firstFilletedPrism1536(t)
	if n := cylinderFaces1494(res1); n != 1 {
		t.Fatalf("after first fillet: %d cylinder faces, want 1", n)
	}
	edge2 := verticalEdgeNearFoot1536(t, res1, 0, 0).ReferenceKey() // opposite (corner-0) edge
	if key := keyString1536(edge2); strings.Contains(key, "fillet:e#") {
		t.Errorf("opposite edge still carries an ordinal name %q — provenance naming did not apply", key)
	}
	f2 := NewDressUpFeatures(fs).AddFillet([][]byte{edge2}, func() float64 { return 1 })
	fs.Recompute()
	if !f2.Health().OK() {
		t.Fatalf("second fillet sick (the #1536 symptom): %+v", f2.Health())
	}
	res2 := fs.Result()[0]
	if r := ops.Validate(res2); !r.Valid || !res2.IsSolid() {
		t.Fatalf("after second fillet: not a valid solid: %+v", r.Issues)
	}
	if n := cylinderFaces1494(res2); n != 2 {
		t.Errorf("after second fillet: %d cylinder faces, want 2 (the first fillet was broken/re-used)", n)
	}
}

// TestFilletEdgesAreProvenanceNamed1536 pins ADR-0043 P1: a fillet's result edges are named by
// their generating PARENTS (the bordering faces / the filleted edge), never a build-order counter.
// No edge carries the old fillet:e#N ordinal, the tangent edges name the filleted edge's blend, and
// every key on the body is distinct (no naming collision).
func TestFilletEdgesAreProvenanceNamed1536(t *testing.T) {
	_, res1 := firstFilletedPrism1536(t)
	keys := map[string]int{}
	tangents := 0
	for _, e := range res1.Edges() {
		k := keyString1536(e.ReferenceKey())
		keys[k]++
		if strings.Contains(k, "fillet:e#") {
			t.Errorf("edge still has a build-order ordinal name: %q", k)
		}
		if strings.Contains(k, "Extrusion1:side-edge#2/fillet:cyl#0") {
			tangents++ // an edge bordering the cylinder blended from the filleted edge
		}
	}
	if tangents == 0 {
		t.Error("no tangent edge names the filleted edge's blend (fillet:cyl) — provenance not applied to the blend")
	}
	for k, c := range keys {
		if c > 1 {
			t.Errorf("naming collision: key %q is shared by %d edges", k, c)
		}
	}
}

// TestFilletNamingIsDeterministic guards reproducibility: two independent builds of the same
// (prism + corner-2 fillet) name the corner-2 tangent edge IDENTICALLY. A reference must resolve to
// the same key on every recompute; flaky naming would lose selections between rebuilds. (Internal
// build-ORDER independence — that the name does not depend on assembleBody's weld/face iteration —
// is guaranteed by NameByParents' canonical ordering, pinned in topo.TestNameByParentsIsOrderIndependent
// and applied to the blend by TestFilletEdgesAreProvenanceNamed1536.)
func TestFilletNamingIsDeterministic(t *testing.T) {
	const tx, ty = 4.6, 3.3 // a tangent edge of corner 2's fillet (cylinder ∩ side face #1)
	_, a := firstFilletedPrism1536(t)
	_, b := firstFilletedPrism1536(t)
	ka := keyString1536(verticalEdgeNearFoot1536(t, a, tx, ty).ReferenceKey())
	kb := keyString1536(verticalEdgeNearFoot1536(t, b, tx, ty).ReferenceKey())
	if ka != kb {
		t.Errorf("non-deterministic fillet edge naming: %q vs %q across identical builds", ka, kb)
	}
	if !strings.Contains(ka, "fillet:x") || strings.Contains(ka, "fillet:e#") {
		t.Errorf("tangent edge key %q is not a provenance name", ka)
	}
}
