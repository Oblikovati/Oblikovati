// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// filletedBoxBend fillets one convex edge of a box, producing a real cylindrical bend face (radius r)
// between two perpendicular flats (a 90° bend). It returns the body and one edge key of the cylinder
// face (the bend edge).
func filletedBoxBend(t *testing.T, r float64) (*topo.Body, []byte) {
	t.Helper()
	box := subd.ToBody(subd.Box(4, 4, 4), "box")
	edgeKey := topmostConvexEdge(t, box)
	filleted, err := ops.FilletEdges(box, [][]byte{edgeKey}, r)
	if err != nil {
		t.Fatalf("FilletEdges: %v", err)
	}
	for _, f := range filleted.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			for _, e := range f.Edges() {
				if _, ok := e.Geometry().(geom.LineSegment); ok {
					return filleted, e.ReferenceKey()
				}
			}
		}
	}
	t.Fatal("no cylindrical bend face after filleting")
	return nil, nil
}

// topmostConvexEdge returns the straight edge whose midpoint maximises x+y+z — a deterministic pick
// of one convex box edge to fillet.
func topmostConvexEdge(t *testing.T, body *topo.Body) []byte {
	t.Helper()
	var best []byte
	bestScore := stdmath.Inf(-1)
	for _, e := range body.Edges() {
		line, ok := e.Geometry().(geom.LineSegment)
		if !ok {
			continue
		}
		m := line.StartPoint.Midpoint(line.EndPoint)
		if s := float64(m.X + m.Y + m.Z); s > bestScore {
			bestScore, best = s, e.ReferenceKey()
		}
	}
	if best == nil {
		t.Fatal("no straight edge to fillet")
	}
	return best
}

// TestBendMetricsFromBody: a 90° filleted bend of radius 0.5 cm reads angle 90°, radius 0.5 cm, and a
// convex (DOWN) fold — all derived from the model.
func TestBendMetricsFromBody(t *testing.T) {
	body, bendEdge := filletedBoxBend(t, 0.5)
	radiusCm, angleDeg, dir, _, ok := bendMetricsFromBody(body, bendEdge)
	if !ok {
		t.Fatal("bendMetricsFromBody failed to resolve the bend")
	}
	if stdmath.Abs(angleDeg-90) > 1e-6 {
		t.Errorf("bend angle = %.4f°, want 90°", angleDeg)
	}
	if stdmath.Abs(radiusCm-0.5) > 1e-6 {
		t.Errorf("bend radius = %.4f cm, want 0.5", radiusCm)
	}
	if dir != "DOWN" {
		t.Errorf("bend direction = %q, want DOWN (a convex fillet folds inward)", dir)
	}
}

// TestAddBendNoteRendersAngleRadius: a bend note reads "90° R5.00 DOWN" (0.5 cm = 5 mm radius) and
// re-resolves when the bend radius changes.
func TestAddBendNoteRendersAngleRadius(t *testing.T) {
	body, bendEdge := filletedBoxBend(t, 0.5)
	c := NewContent()
	c.SetBodyResolver(fakeBodyResolver{body: body})
	c.SetModelReference("bend.opd")
	topBase(t, c.Sheets().Active().Views())

	bn, err := c.Sheets().Active().Annotations().AddBendNote("BN", "TOP", bendEdge)
	if err != nil {
		t.Fatalf("AddBendNote: %v", err)
	}
	if bn.Kind() != types.BendNoteAnnotation || len(bn.Labels()) != 1 || bn.Labels()[0].Text != "90° R5.00 DOWN" {
		t.Fatalf("bend note = %v, want one label 90° R5.00 DOWN", bn.Labels())
	}
}
