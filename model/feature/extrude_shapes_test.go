// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Extrusion-shape coverage for complex closed paths — arcs, ellipses, splines, and
// their mixtures — regression for the bug where loops with arcs flattened to chords
// (so the prism enclosed no/incorrect volume). Each case asserts the solid is
// watertight, manifold, correctly oriented, has the expected bounding box, and that
// its volume equals the cross-section polygon's area times the extrude height (the
// prism faithfully follows the sampled curve, not a chord).

// extrudeProfile extrudes profile profileIndex of sk by height as a new body and
// returns the resulting solid, failing the test if it goes sick.
func extrudeProfile(t *testing.T, sk *sketch.Sketch, profileIndex int, height float64) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	pf := NewExtrudeFeatures(fs).AddByDistanceExtent(sk, profileIndex, ops.NewBody, func() float64 { return height })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("extrude went sick: %+v", pf.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("result has %d bodies, want 1", len(bodies))
	}
	return bodies[0]
}

// assertWatertightSolid asserts the body is a closed, manifold, consistently oriented
// solid (the invariants every extruded prism must hold).
func assertWatertightSolid(t *testing.T, b *topo.Body) {
	t.Helper()
	if !b.IsSolid() {
		t.Error("extrude result is not a solid")
	}
	if open := ops.BoundaryEdges(b); len(open) != 0 {
		t.Errorf("solid has %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(b); !r.Valid {
		t.Errorf("solid failed validation: %+v", r.Issues)
	}
}

// assertPrismVolume asserts the mesh volume equals the cross-section area times the
// height — i.e. the prism is exactly the swept profile polygon, arcs and all.
func assertPrismVolume(t *testing.T, b *topo.Body, sk *sketch.Sketch, profileIndex int, height float64) {
	t.Helper()
	poly := sk.Profiles().Item(profileIndex).OuterLoop().Polygon()
	want := polygonArea(poly) * height
	got := meshVolume(b)
	if rel := stdmath.Abs(got-want) / want; rel > 1e-3 {
		t.Errorf("prism volume = %.6f, want %.6f (cross-section %.6f × height %.1f)", got, want, polygonArea(poly), height)
	}
}

func TestExtrudeHalfDiscFromArcAndLine(t *testing.T) {
	const r, h = 5.0, 4.0
	sk := halfDiscSketch(r)
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)
	assertPrismVolume(t, b, sk, 0, h)
	// The faceted half-disc area is within a few percent of the analytic πr²/2.
	if a := polygonArea(sk.Profiles().Item(0).OuterLoop().Polygon()); relErr(a, stdmath.Pi*r*r/2) > 0.02 {
		t.Errorf("half-disc cross-section area = %.4f, want ≈ %.4f", a, stdmath.Pi*r*r/2)
	}
}

func TestExtrudeStadiumFromTwoArcsTwoLines(t *testing.T) {
	const l, r, h = 10.0, 3.0, 4.0
	sk := stadiumSketch(l, r)
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)
	assertPrismVolume(t, b, sk, 0, h)
	box := b.RangeBox().Diagonal()
	if !approxEq(box.X, l+2*r) || !approxEq(box.Y, 2*r) || !approxEq(box.Z, h) {
		t.Errorf("stadium box = %v, want (%.1f, %.1f, %.1f)", box, l+2*r, 2*r, h)
	}
	if a := polygonArea(sk.Profiles().Item(0).OuterLoop().Polygon()); relErr(a, 2*r*l+stdmath.Pi*r*r) > 0.02 {
		t.Errorf("stadium cross-section area = %.4f, want ≈ %.4f", a, 2*r*l+stdmath.Pi*r*r)
	}
}

func TestExtrudeRoundedRectangleFilletedCorners(t *testing.T) {
	const w, ht, r, h = 12.0, 8.0, 2.0, 5.0
	sk := roundedRectSketch(w, ht, r)
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)
	assertPrismVolume(t, b, sk, 0, h)
	box := b.RangeBox().Diagonal()
	if !approxEq(box.X, w) || !approxEq(box.Y, ht) || !approxEq(box.Z, h) {
		t.Errorf("rounded-rect box = %v, want (%.1f, %.1f, %.1f)", box, w, ht, h)
	}
	// Area = full rectangle minus the four corners the fillets round off: 4r² − πr².
	want := w*ht - (4-stdmath.Pi)*r*r
	if a := polygonArea(sk.Profiles().Item(0).OuterLoop().Polygon()); relErr(a, want) > 0.02 {
		t.Errorf("rounded-rect area = %.4f, want ≈ %.4f", a, want)
	}
}

func TestExtrudeEllipse(t *testing.T) {
	const a, bb, h = 6.0, 3.0, 4.0
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), a, bb)
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)
	assertPrismVolume(t, b, sk, 0, h)
	box := b.RangeBox().Diagonal()
	if !approxEq(box.X, 2*a) || !approxEq(box.Y, 2*bb) || !approxEq(box.Z, h) {
		t.Errorf("ellipse box = %v, want (%.1f, %.1f, %.1f)", box, 2*a, 2*bb, h)
	}
	if ar := polygonArea(sk.Profiles().Item(0).OuterLoop().Polygon()); relErr(ar, stdmath.Pi*a*bb) > 0.02 {
		t.Errorf("ellipse area = %.4f, want ≈ %.4f", ar, stdmath.Pi*a*bb)
	}
}

func TestExtrudeClosedSplineBlob(t *testing.T) {
	const h = 4.0
	// A closed fit-spline through points roughly on a circle of radius 5 → a smooth
	// blob enclosing a positive area near πr².
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Splines().AddByPoints(ringPoints(6, 5), true)
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)
	assertPrismVolume(t, b, sk, 0, h)
	if a := polygonArea(sk.Profiles().Item(0).OuterLoop().Polygon()); a < 60 || a > 90 {
		t.Errorf("closed-spline blob area = %.4f, want roughly π·5² ≈ 78.5", a)
	}
}

func TestExtrudeMixedLinesArcSpline(t *testing.T) {
	const h = 3.0
	sk := mixedLineArcSplineSketch()
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)
	assertPrismVolume(t, b, sk, 0, h)
}

func TestExtrudeTwoComplexClosedPathsAsSeparateBodies(t *testing.T) {
	// "One or more closed paths": a sketch holding two disjoint complex loops yields
	// two profiles; each extrudes independently into its own valid solid.
	const h = 4.0
	sk := stadiumSketch(8, 2)
	sk.Ellipses().Add(math.P2(30, 0), math.V2(1, 0), 4, 2) // a second, disjoint region
	if n := sk.Profiles().Count(); n != 2 {
		t.Fatalf("sketch has %d profiles, want 2 disjoint closed paths", n)
	}
	for i := 0; i < 2; i++ {
		b := extrudeProfile(t, sk, i, h)
		assertWatertightSolid(t, b)
		assertPrismVolume(t, b, sk, i, h)
	}
}

func TestExtrudeMultipleRegionsMergeIntoOneBody(t *testing.T) {
	// Two disjoint square regions in one sketch, extruded together as one feature →
	// a single body whose volume is the sum of the two prisms.
	const side, gap, h = 2.0, 10.0, 4.0
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	addSquareAt(sk, 0, side)
	addSquareAt(sk, gap, side)
	if n := sk.Profiles().Count(); n != 2 {
		t.Fatalf("sketch has %d regions, want 2", n)
	}
	fs := NewPartFeatures(nil, nil)
	pf := NewExtrudeFeatures(fs).AddByDistanceExtentProfiles(sk, []int{0, 1}, ops.NewBody, func() float64 { return h })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("multi-region extrude went sick: %+v", pf.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("result has %d bodies, want 1 merged", len(fs.Result()))
	}
	b := fs.Result()[0]
	assertWatertightSolid(t, b)
	if got, want := meshVolume(b), 2*side*side*h; relErr(got, want) > 1e-3 {
		t.Errorf("merged volume = %.4f, want %.4f (two %g×%g prisms)", got, want, side, side)
	}
}

func TestExtrudeSplitRegionsRebuildWholeShape(t *testing.T) {
	// A rectangle split by a line into two regions; extruding both regions together
	// reconstitutes the full rectangle's prism (regression for the reported bug).
	const w, ht, h = 10.0, 6.0, 3.0
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	addSquareRect(sk, 0, 0, w, ht)
	sk.Lines().Add(sk.Points().Add(math.P2(0, 3)), sk.Points().Add(math.P2(w, 3))) // divider
	if n := sk.Profiles().Count(); n != 2 {
		t.Fatalf("split rectangle has %d regions, want 2", n)
	}
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtentProfiles(sk, []int{0, 1}, ops.NewBody, func() float64 { return h })
	fs.Recompute()
	b := fs.Result()[0]
	assertWatertightSolid(t, b)
	if got, want := meshVolume(b), w*ht*h; relErr(got, want) > 1e-3 {
		t.Errorf("split-then-merge volume = %.4f, want %.4f (whole rectangle)", got, want)
	}
}

// --- shape builders ------------------------------------------------------------

// addSquareRect adds a closed rectangle [x0,y0]–[x1,y1] sharing corner points.
func addSquareRect(s *sketch.Sketch, x0, y0, x1, y1 float64) {
	c00 := s.Points().Add(math.P2(x0, y0))
	c10 := s.Points().Add(math.P2(x1, y0))
	c11 := s.Points().Add(math.P2(x1, y1))
	c01 := s.Points().Add(math.P2(x0, y1))
	s.Lines().Add(c00, c10)
	s.Lines().Add(c10, c11)
	s.Lines().Add(c11, c01)
	s.Lines().Add(c01, c00)
}

// addSquareAt adds a side×side square with its lower-left corner at (dx,0).
func addSquareAt(s *sketch.Sketch, dx, side float64) { addSquareRect(s, dx, 0, dx+side, side) }

// halfDiscSketch is a diameter line closed by a semicircular arc of radius r.
func halfDiscSketch(r float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	left := s.Points().Add(math.P2(-r, 0))
	right := s.Points().Add(math.P2(r, 0))
	s.Lines().Add(right, left)
	s.Arcs().Add(s.Points().Add(math.P2(0, 0)), left, right, true)
	return s
}

// stadiumSketch is a discorectangle: two horizontal lines of length l joined by two
// semicircular end caps of radius r (left cap centered at the origin).
func stadiumSketch(l, r float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	tl := s.Points().Add(math.P2(0, r))
	tr := s.Points().Add(math.P2(l, r))
	br := s.Points().Add(math.P2(l, -r))
	bl := s.Points().Add(math.P2(0, -r))
	s.Lines().Add(tl, tr)
	s.Arcs().Add(s.Points().Add(math.P2(l, 0)), tr, br, false) // right cap, through (l+r,0)
	s.Lines().Add(br, bl)
	s.Arcs().Add(s.Points().Add(math.P2(0, 0)), bl, tl, false) // left cap, through (-r,0)
	return s
}

// roundedRectSketch is a w×ht rectangle centered at the origin with its four corners
// replaced by quarter-circle fillets of radius r (four lines + four arcs).
func roundedRectSketch(w, ht, r float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	hw, hh := w/2, ht/2
	// Tangent points where each straight edge meets a corner fillet.
	rt := s.Points().Add(math.P2(hw, hh-r))  // right edge, top end
	rb := s.Points().Add(math.P2(hw, -hh+r)) // right edge, bottom end
	bl := s.Points().Add(math.P2(-hw+r, -hh))
	br := s.Points().Add(math.P2(hw-r, -hh))
	lt := s.Points().Add(math.P2(-hw, hh-r))
	lb := s.Points().Add(math.P2(-hw, -hh+r))
	tl := s.Points().Add(math.P2(-hw+r, hh))
	tr := s.Points().Add(math.P2(hw-r, hh))
	s.Lines().Add(rb, rt)                                             // right edge
	s.Arcs().Add(s.Points().Add(math.P2(hw-r, hh-r)), rt, tr, true)   // top-right fillet
	s.Lines().Add(tr, tl)                                             // top edge
	s.Arcs().Add(s.Points().Add(math.P2(-hw+r, hh-r)), tl, lt, true)  // top-left fillet
	s.Lines().Add(lt, lb)                                             // left edge
	s.Arcs().Add(s.Points().Add(math.P2(-hw+r, -hh+r)), lb, bl, true) // bottom-left fillet
	s.Lines().Add(bl, br)                                             // bottom edge
	s.Arcs().Add(s.Points().Add(math.P2(hw-r, -hh+r)), br, rb, true)  // bottom-right fillet
	return s
}

// mixedLineArcSplineSketch is a closed loop made of three different curve kinds: a
// base line, a semicircular arc on the right, and a fit spline arcing back to start.
func mixedLineArcSplineSketch() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(10, 0))
	c := s.Points().Add(math.P2(10, 6))
	s.Lines().Add(a, b)                                      // base
	s.Arcs().Add(s.Points().Add(math.P2(10, 3)), b, c, true) // right semicircle bulge
	s.Splines().AddWithPoints([]*sketch.Point{               // spline lid back to start
		c,
		s.Points().Add(math.P2(6, 9)),
		s.Points().Add(math.P2(2, 8)),
		a,
	}, false, true)
	return s
}

// ringPoints returns n points evenly spaced on a circle of radius r, for a closed
// spline that traces a smooth blob.
func ringPoints(n int, r float64) []math.Point2 {
	pts := make([]math.Point2, n)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		pts[i] = math.P2(r*stdmath.Cos(a), r*stdmath.Sin(a))
	}
	return pts
}

// --- measurement helpers -------------------------------------------------------

// polygonArea is the absolute shoelace area of a closed polygon.
func polygonArea(poly []math.Point2) float64 {
	a := 0.0
	for i, n := 0, len(poly); i < n; i++ {
		j := (i + 1) % n
		a += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	return stdmath.Abs(a) / 2
}

// meshVolume returns the enclosed volume of a closed triangle mesh via the signed
// tetrahedron sum (divergence theorem). Each triangle is first oriented outward using
// its vertex normals, so the sum is correct regardless of the tessellator's winding.
func meshVolume(b *topo.Body) float64 {
	mesh, _ := ops.TessellateBody(b, ops.DefaultQuality())
	vol := 0.0
	for t := 0; t < mesh.TriangleCount(); t++ {
		i, j, k := mesh.Indices[3*t], mesh.Indices[3*t+1], mesh.Indices[3*t+2]
		p0, p1, p2 := mesh.Positions[i], mesh.Positions[j], mesh.Positions[k]
		if p0.VectorTo(p1).Cross(p0.VectorTo(p2)).Dot(mesh.Normals[i]) < 0 {
			p1, p2 = p2, p1 // flip to outward winding
		}
		vol += p0.AsVector().Dot(p1.AsVector().Cross(p2.AsVector()))
	}
	return vol / 6
}

func relErr(got, want float64) float64 { return stdmath.Abs(got-want) / stdmath.Abs(want) }
