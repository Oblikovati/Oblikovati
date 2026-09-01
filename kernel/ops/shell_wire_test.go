// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"bytes"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// cavityBody is a 4×4×4 cube with a centered 2×2×2 cavity — the canonical
// two-shell solid (#629): the outer skin plus the void skin.
func cavityBody(t *testing.T) *topo.Body {
	t.Helper()
	big := tetraBox(t, math.P3(0, 0, 0), 4)
	small := tetraBox(t, math.P3(1, 1, 1), 2)
	res, err := Boolean(Cut, big, small)
	if err != nil {
		t.Fatalf("cavity cut: %v", err)
	}
	return res
}

func tetraBox(t *testing.T, p math.Point3, s float64) *topo.Body {
	t.Helper()
	m := subd.Box(s, s, s)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(p.AsVector())
	}
	return subd.ToBody(m, "box")
}

// TestCavityBodyHasVoidShell: the cut regroups into two closed shells; the
// inner one classifies as a void with negative signed volume ≈ −8.
func TestCavityBodyHasVoidShell(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	shells := body.Shells()
	if len(shells) != 2 {
		t.Fatalf("cavity body has %d shells, want 2 (outer + void)", len(shells))
	}
	voids := 0
	for _, sh := range shells {
		if !sh.IsClosed() {
			t.Errorf("shell %d not closed", sh.Index())
		}
		if ShellIsVoidInBody(body, sh) {
			voids++
			if v := query.ShellSignedVolume(sh, DefaultQuality()); stdmath.Abs(v+8) > 0.05 {
				t.Errorf("void shell signed volume = %g, want ~-8", v)
			}
		}
	}
	if voids != 1 {
		t.Errorf("cavity body classifies %d void shells, want 1", voids)
	}
}

// TestShellKeysAndRangeBoxes: shells carry distinct reference keys (kind-
// prefixed) and connectivity-true range boxes.
func TestShellKeysAndRangeBoxes(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	a, b := body.Shells()[0], body.Shells()[1]
	if bytes.Equal(a.ReferenceKey(), b.ReferenceKey()) {
		t.Error("the two shells must carry distinct reference keys")
	}
	if a.ReferenceKey()[0] != byte(topo.KindShell) {
		t.Error("shell key must be kind-prefixed")
	}
	outer, void := a, b
	if ShellIsVoidInBody(body, a) {
		outer, void = b, a
	}
	vd := float64(void.RangeBox().Diagonal().Length())
	od := float64(outer.RangeBox().Diagonal().Length())
	if vd >= od {
		t.Errorf("void shell box diagonal %g should be smaller than the outer %g", vd, od)
	}
	if len(outer.Edges()) == 0 || len(void.Edges()) == 0 {
		t.Error("shells must report their edges")
	}
}

// TestShellContainmentVerdicts: a point in the material is inside the outer
// shell region but the body classifies cavity points as outside material.
func TestShellContainmentVerdicts(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	q := DefaultQuality()
	if c := query.BodyContainment(body, math.P3(0.5, 0.5, 0.5), q, 1e-6); c != query.ContainInside {
		t.Errorf("material point = %v, want inside", c)
	}
	if c := query.BodyContainment(body, math.P3(2, 2, 2), q, 1e-6); c != query.ContainOutside {
		t.Errorf("cavity center = %v, want outside (no material there)", c)
	}
	if c := query.BodyContainment(body, math.P3(0, 2, 2), q, 1e-6); c != query.ContainOn {
		t.Errorf("outer wall point = %v, want on", c)
	}
	if c := query.BodyContainment(body, math.P3(10, 0, 0), q, 1e-6); c != query.ContainOutside {
		t.Errorf("far point = %v, want outside", c)
	}
}

// squareWireBody attaches a unit-square wire (CCW in the XY plane) to a body.
func squareWireBody(side float64) (*topo.Body, *topo.Wire) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("w", "body", 0)))
	p := []math.Point3{
		math.P3(0, 0, 0), math.P3(math.Scalar(side), 0, 0),
		math.P3(math.Scalar(side), math.Scalar(side), 0), math.P3(0, math.Scalar(side), 0),
	}
	v := make([]*topo.Vertex, 4)
	for i := range p {
		v[i] = bld.AddVertex(p[i], topo.NewLineage(topo.Tok("w", "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[j]), v[i], v[j], topo.NewLineage(topo.Tok("w", "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	body := bld.Build()
	w := body.AttachWire(topo.NewLineage(topo.Tok("w", "wire", 0)), uses)
	return body, w
}

// TestWireBasics: closure, planarity, plane frame, keys.
func TestWireBasics(t *testing.T) {
	t.Parallel()
	body, w := squareWireBody(1)
	if len(body.Wires()) != 1 {
		t.Fatalf("body has %d wires, want 1", len(body.Wires()))
	}
	if !w.IsClosed() || !w.IsPlanar() {
		t.Errorf("unit square wire: closed=%v planar=%v, want both true", w.IsClosed(), w.IsPlanar())
	}
	_, n, ok := w.PlaneFrame()
	if !ok || stdmath.Abs(stdmath.Abs(float64(n.Z))-1) > 1e-9 {
		t.Errorf("plane normal = %v (ok=%v), want ±Z", n, ok)
	}
	if w.ReferenceKey()[0] != byte(topo.KindWire) {
		t.Error("wire key must be kind-prefixed")
	}
	if len(w.Edges()) != 4 {
		t.Errorf("wire has %d edges, want 4", len(w.Edges()))
	}
}

// TestWireNonPlanarDetected: lifting one vertex off the plane kills planarity.
func TestWireNonPlanarDetected(t *testing.T) {
	t.Parallel()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("w", "body", 0)))
	p := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0.5), math.P3(0, 1, 0)}
	v := make([]*topo.Vertex, 4)
	for i := range p {
		v[i] = bld.AddVertex(p[i], topo.NewLineage(topo.Tok("w", "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(p[i], p[j]), v[i], v[j], topo.NewLineage(topo.Tok("w", "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	w := bld.Build().AttachWire(topo.NewLineage(topo.Tok("w", "wire", 0)), uses)
	if w.IsPlanar() {
		t.Error("a twisted quad chain must not be planar")
	}
}

// wireLength sums the wire's edge curve lengths by dense sampling.
func wireLength(w *topo.Wire) float64 {
	sum := 0.0
	for _, u := range w.Uses() {
		c := u.Edge.Geometry()
		lo, hi := c.Domain()
		prev := c.PointAt(lo)
		for i := 1; i <= 64; i++ {
			p := c.PointAt(lo + (hi-lo)*float64(i)/64)
			sum += float64(prev.DistanceTo(p))
			prev = p
		}
	}
	return sum
}

// TestOffsetSquareInwardTrims: offsetting the CCW unit square LEFT (inward)
// trims the corners — four lines, perimeter 4(1−2d).
func TestOffsetSquareInwardTrims(t *testing.T) {
	t.Parallel()
	_, w := squareWireBody(1)
	out, err := OffsetPlanarWire(w, math.V3(0, 0, 1), 0.1, WireCornerLinear)
	if err != nil {
		t.Fatalf("OffsetPlanarWire: %v", err)
	}
	ow := out.Wires()[0]
	if len(ow.Edges()) != 4 {
		t.Fatalf("inward offset has %d edges, want 4 (pure trim)", len(ow.Edges()))
	}
	if l := wireLength(ow); stdmath.Abs(l-4*0.8) > 1e-9 {
		t.Errorf("inward offset perimeter = %g, want %g", l, 4*0.8)
	}
}

// TestOffsetSquareOutwardCircular: offsetting RIGHT (outward) with circular
// closure rounds each corner — 8 edges, perimeter 4 + 2π·d.
func TestOffsetSquareOutwardCircular(t *testing.T) {
	t.Parallel()
	_, w := squareWireBody(1)
	out, err := OffsetPlanarWire(w, math.V3(0, 0, 1), -0.25, WireCornerCircular)
	if err != nil {
		t.Fatalf("OffsetPlanarWire: %v", err)
	}
	ow := out.Wires()[0]
	if len(ow.Edges()) != 8 {
		t.Fatalf("outward circular offset has %d edges, want 8 (4 lines + 4 arcs)", len(ow.Edges()))
	}
	want := 4 + 2*stdmath.Pi*0.25
	// wireLength samples 64 chords/edge; each arc reads ~2.5e-5 relative short.
	if l := wireLength(ow); stdmath.Abs(l-want) > 1e-4 {
		t.Errorf("outward offset perimeter = %g, want %g", l, want)
	}
}

// TestOffsetSquareOutwardLinear: linear closure miters each corner — the
// perimeter of the (1+2d) square.
func TestOffsetSquareOutwardLinear(t *testing.T) {
	t.Parallel()
	_, w := squareWireBody(1)
	out, err := OffsetPlanarWire(w, math.V3(0, 0, 1), -0.25, WireCornerLinear)
	if err != nil {
		t.Fatalf("OffsetPlanarWire: %v", err)
	}
	want := 4 * 1.5
	if l := wireLength(out.Wires()[0]); stdmath.Abs(l-want) > 1e-9 {
		t.Errorf("mitered outward perimeter = %g, want %g", l, want)
	}
}

// circleWireBody is a single full-circle arc wire of radius r in XY.
func circleWireBody(r float64) *topo.Wire {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("c", "body", 0)))
	arc, _ := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, 0, 2*stdmath.Pi)
	v := bld.AddVertex(math.P3(math.Scalar(r), 0, 0), topo.NewLineage(topo.Tok("c", "vertex", 0)))
	e := bld.AddEdge(arc, v, v, topo.NewLineage(topo.Tok("c", "edge", 0)))
	return bld.Build().AttachWire(topo.NewLineage(topo.Tok("c", "wire", 0)), []topo.Use{topo.Fwd(e)})
}

// TestOffsetCircleRadiusShift: a CCW circle offset left (toward center)
// shrinks the radius; the collapse case errors with the offending values.
func TestOffsetCircleRadiusShift(t *testing.T) {
	t.Parallel()
	w := circleWireBody(2)
	out, err := OffsetPlanarWire(w, math.V3(0, 0, 1), 0.5, WireCornerCircular)
	if err != nil {
		t.Fatalf("OffsetPlanarWire: %v", err)
	}
	want := 2 * stdmath.Pi * 1.5
	// chord sampling reads the circle ~2.4e-4 short at 64 segments.
	if l := wireLength(out.Wires()[0]); stdmath.Abs(l-want) > 1e-2 {
		t.Errorf("offset circle length = %g, want %g", l, want)
	}
	if _, err := OffsetPlanarWire(w, math.V3(0, 0, 1), 2.5, WireCornerCircular); err == nil {
		t.Error("an offset past the radius must error (arc collapse)")
	}
}

// TestOffsetRejectsOutOfPlane: a wire that leaves the stated plane errors
// precisely rather than silently projecting.
func TestOffsetRejectsOutOfPlane(t *testing.T) {
	t.Parallel()
	_, w := squareWireBody(1)
	if _, err := OffsetPlanarWire(w, math.V3(1, 0, 0), 0.1, WireCornerLinear); err == nil {
		t.Error("offsetting an XY wire in a YZ plane must error")
	}
}
