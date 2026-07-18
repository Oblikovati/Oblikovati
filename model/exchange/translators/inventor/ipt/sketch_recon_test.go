// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestReadInlineLineGeometry verifies a line node's inline (midpoint, unit-direction) decode on the
// known corpus: k_base's L1 (0,0)-(3,0) has midpoint (1.5,0) and unit direction (1,0); L2 (7,0)-(7,4)
// has midpoint (7,2) and direction (0,1). collectItems populates these via readInlineLine, so the
// infinite line is recovered without resolving any point reference.
func TestReadInlineLineGeometry(t *testing.T) {
	seg := segFor(t, "k_base.ipt")
	want := map[[2]int64][4]float64{ // key = rounded (mid.x,mid.y) → mid.x,mid.y,dir.x,dir.y
		{15000, 0}:      {1.5, 0, 1, 0},      // L1 (0,0)-(3,0)
		{70000, 20000}:  {7, 2, 0, 1},        // L2 (7,0)-(7,4)
		{115000, 20000}: {11.5, 2, 0.6, 0.8}, // Ldiag (10,0)-(13,4), dir (3,4)/5
	}
	got := map[[2]int64]bool{}
	for _, it := range collectItems(seg) {
		if it.line == nil || !it.line.unit {
			continue
		}
		k := [2]int64{r4(it.line.mid.X), r4(it.line.mid.Y)}
		w, ok := want[k]
		if !ok {
			continue
		}
		got[k] = true
		l := it.line
		if absf(l.mid.X-w[0]) > 1e-6 || absf(l.mid.Y-w[1]) > 1e-6 || absf(l.dir.X-w[2]) > 1e-6 || absf(l.dir.Y-w[3]) > 1e-6 {
			t.Errorf("line mid=%v dir=%v, want mid(%.3g,%.3g) dir(%.3g,%.3g)", l.mid, l.dir, w[0], w[1], w[2], w[3])
		}
	}
	for k := range want {
		if !got[k] {
			t.Errorf("no unit inline line found for midpoint key %v", k)
		}
	}
}

// TestReconstructInlineLoop verifies the geometry-first reconstruction: a rectangle's four inline
// lines with their four corner points reconstruct to the exact rectangle (Resolved), while polluting
// the point set (so a corner has no matching point) is rejected — the self-validation gate.
func TestReconstructInlineLoop(t *testing.T) {
	// unit square, corners (±1,±1)
	rect := []sketchItem{
		{line: &lineRefs{mid: Point2D{0, 1}, dir: Point2D{1, 0}, unit: true}},  // top
		{line: &lineRefs{mid: Point2D{1, 0}, dir: Point2D{0, 1}, unit: true}},  // right
		{line: &lineRefs{mid: Point2D{0, -1}, dir: Point2D{1, 0}, unit: true}}, // bottom
		{line: &lineRefs{mid: Point2D{-1, 0}, dir: Point2D{0, 1}, unit: true}}, // left
		{pt: &idPoint{p: Point2D{1, 1}}}, {pt: &idPoint{p: Point2D{1, -1}}},
		{pt: &idPoint{p: Point2D{-1, -1}}}, {pt: &idPoint{p: Point2D{-1, 1}}},
	}
	s, ok := reconstructInlineLoop(rect)
	if !ok {
		t.Fatal("rectangle with matching corner points should reconstruct")
	}
	if !s.Resolved || len(s.Lines) != 4 || len(s.Points) != 4 {
		t.Fatalf("reconstructed sketch = %d lines %d pts resolved=%v, want 4/4/true", len(s.Lines), len(s.Points), s.Resolved)
	}
	for _, c := range s.Points {
		if absf(absf(c.X)-1) > 1e-6 || absf(absf(c.Y)-1) > 1e-6 {
			t.Errorf("corner %v is not a unit-square corner", c)
		}
	}
	// Remove the matching points → every corner fails the point gate → reject.
	noPts := rect[:4]
	if _, ok := reconstructInlineLoop(noPts); ok {
		t.Error("reconstruction without matching corner points must be rejected")
	}
	// An arc in the cluster takes it out of scope (line-only).
	withArc := append(append([]sketchItem{}, rect...), sketchItem{arc: &arcEnt{radius: 0.5}})
	if _, ok := reconstructInlineLoop(withArc); ok {
		t.Error("a cluster containing an arc must fall through (line-only scope)")
	}
	// Construction lines are dropped: adding a construction diagonal must not break the rectangle.
	withConstr := append(append([]sketchItem{}, rect...),
		sketchItem{line: &lineRefs{mid: Point2D{0, 0}, dir: Point2D{0.7071, 0.7071}, unit: true, constr: true}})
	if s2, ok := reconstructInlineLoop(withConstr); !ok || len(s2.Lines) != 4 {
		t.Errorf("construction line should be dropped, still reconstructing 4 lines; got ok=%v lines=%d", ok, len(s2.Lines))
	}
}
