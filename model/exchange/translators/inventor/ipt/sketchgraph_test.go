// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// TestGraphSketchesExact pins the graph decode against parts whose sketches are known from how the
// corpus was authored: 14_box_two draws Rect(0,0,4,2) then Rect(1,0.5,3,1.5), one per sketch.
func TestGraphSketchesExact(t *testing.T) {
	got := GraphSketches(openDoc(t, "14_box_two.ipt"))
	if len(got) != 2 {
		t.Fatalf("got %d sketches, want 2 (the file has two Sketch2D nodes)", len(got))
	}
	for i, want := range [][2]Point2D{
		{{0, 0}, {4, 2}}, // rect corners: min, max
		{{1, 0.5}, {3, 1.5}},
	} {
		s := got[i]
		if len(s.Lines) != 4 || len(s.Points) != 4 {
			t.Errorf("sketch %d: %d lines / %d points, want 4 / 4", i, len(s.Lines), len(s.Points))
			continue
		}
		if !s.Resolved {
			t.Errorf("sketch %d: Resolved=false, but a graph decode is exact", i)
		}
		lo, hi := boundsOf(s.Lines)
		if !nearPt(lo, want[0]) || !nearPt(hi, want[1]) {
			t.Errorf("sketch %d spans %v..%v, want %v..%v", i, lo, hi, want[0], want[1])
		}
		if !closedLoop(s.Lines) {
			t.Errorf("sketch %d: lines do not form a closed loop", i)
		}
	}
}

// TestGraphSketchesDeclineWithoutSketchNodes keeps the caller's fallback honest: a part with no
// Sketch2D node must yield nothing rather than an empty shell.
func TestGraphSketchesDeclineWithoutSketchNodes(t *testing.T) {
	if got := GraphSketches(openDoc(t, "blank_a.ipt")); len(got) != 0 {
		t.Errorf("blank part decoded %d sketches, want 0", len(got))
	}
}

// TestGraphSketchesEllipse guards ellipse decoding through the graph path.
func TestGraphSketchesEllipse(t *testing.T) {
	total := 0
	for _, s := range GraphSketches(openDoc(t, "ke_ellipse.ipt")) {
		total += len(s.Ellipses)
	}
	if total != 1 {
		t.Errorf("got %d ellipses, want 1", total)
	}
}

func boundsOf(ls []Line) (lo, hi Point2D) {
	lo, hi = ls[0].A, ls[0].A
	for _, l := range ls {
		for _, p := range []Point2D{l.A, l.B} {
			lo.X, lo.Y = minf(lo.X, p.X), minf(lo.Y, p.Y)
			hi.X, hi.Y = maxf(hi.X, p.X), maxf(hi.Y, p.Y)
		}
	}
	return lo, hi
}

// closedLoop reports whether every endpoint is shared by exactly two lines — the exact topology the
// graph decode claims, which the cluster decode could only guess at.
func closedLoop(ls []Line) bool {
	deg := map[[2]int64]int{}
	for _, l := range ls {
		deg[endpointKey(l.A)]++
		deg[endpointKey(l.B)]++
	}
	for _, d := range deg {
		if d != 2 {
			return false
		}
	}
	return len(deg) == len(ls)
}

func nearPt(a, b Point2D) bool { return absf(a.X-b.X) < 1e-9 && absf(a.Y-b.Y) < 1e-9 }

// TestGraphSketchesMultiPointEdges guards the endpoint rule against a real part whose edges carry
// MORE than two point references. An edge lists every point lying on it — a midpoint, a coincidence
// — and only the first two are its ends (InventorLoader's addSketch_Line2D reads points[0] and
// points[1]). Requiring exactly two silently DROPPED such an edge, which cost the sketch a curve and
// nulled the region built from it. This fixture is ReelToReel's TorquimeterDisk: 4 of its 15 edges
// have 3+ point refs, and none may be lost.
func TestGraphSketchesMultiPointEdges(t *testing.T) {
	d := openDoc(t, "real_multipoint_disk.ipt")
	nodes := dcNodes(d)
	_, index := sketchOrdinals(nodes)

	declared, multi := 0, 0
	for _, n := range nodes {
		if n.typ != line2DNodeType && n.typ != arc2DNodeType {
			continue
		}
		if _, ok := entityOwner(n, index); !ok {
			continue
		}
		declared++
		if refs, _, ok := refList2(n.payload, edgePointsListOffset); ok && len(refs) > 2 {
			multi++
		}
	}
	if multi == 0 {
		t.Fatal("fixture no longer has an edge with >2 point refs — it cannot guard the rule")
	}
	decoded := 0
	for _, s := range GraphSketches(d) {
		decoded += len(s.Lines) + len(s.Arcs)
	}
	if decoded != declared {
		t.Errorf("decoded %d line/arc curves from %d the file declares — %d dropped; an edge with "+
			"more than two point refs must keep its FIRST TWO as endpoints, not be discarded",
			decoded, declared, declared-decoded)
	}
}
