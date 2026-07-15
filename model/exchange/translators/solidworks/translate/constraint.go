// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"math"

	m "oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/solidworks/sldprt"
	"oblikovati.org/model/sketch"
)

// geomEps is the tolerance (cm) for recognising that a relation already holds in the emitted
// geometry (a line is horizontal, two lines are tangent, …).
const geomEps = 1e-6

// applyConstraints binds the decoded geometric relations onto the emitted sketch. SolidWorks stores
// a relation as a type plus entity references; the references need the MFC object-graph walk to
// resolve, so — as in the Inventor translator — each relation is instead applied to the geometry
// that already satisfies it, up to the decoded count and only while the sketch has free degrees of
// freedom. Because the geometry is exact, applying a satisfied relation removes DOF without moving a
// point. Coincidence is omitted: touching corners already share one point (see sharedPoints).
func applyConstraints(sk *sketch.Sketch, cons []sldprt.Constraint, lines []*sketch.Line, arcs []*sketch.Arc, circles []*sketch.Circle, points []*sketch.Point) {
	n := map[sldprt.ConstraintKind]int{}
	for _, c := range cons {
		if c.Kind.IsGeometric() {
			n[c.Kind]++
		}
	}
	g := sk.GeometricConstraints()
	applyMidpoints(sk, n, points, lines, g)
	applySymmetric(sk, n, lines, g)
	applyToLines(sk, n, sldprt.Horizontal, lines, func(l *sketch.Line) bool {
		if !lineHorizontal(l) {
			return false
		}
		g.AddHorizontal(l.A, l.B)
		return true
	})
	applyToLines(sk, n, sldprt.Vertical, lines, func(l *sketch.Line) bool {
		if !lineVertical(l) {
			return false
		}
		g.AddVertical(l.A, l.B)
		return true
	})
	applyToLinePairs(sk, n, sldprt.Perpendicular, lines, perpendicular, g.AddPerpendicular)
	applyToLinePairs(sk, n, sldprt.Parallel, lines, parallel, g.AddParallel)
	applyToLinePairs(sk, n, sldprt.Collinear, lines, collinear, g.AddCollinear)
	applyToLinePairs(sk, n, sldprt.EqualLength, lines, equalLength, g.AddEqualLength)
	applyTangents(sk, n, lines, curvesOf(arcs, circles), g)
	applyToCurvePairs(sk, n, sldprt.Concentric, curvesOf(arcs, circles), concentric, g.AddConcentric)
}

// applySymmetric binds each decoded symmetry relation to a pair of lines that are already mirror
// images about a construction line (the axis): the two lines' corresponding endpoints are made
// symmetric about that axis. SolidWorks stores line symmetry; Oblikovati expresses it per point, so
// each matched pair yields two point-symmetry constraints. Applied up to the decoded count while DOF
// remains; because the geometry is already symmetric, no point moves.
func applySymmetric(sk *sketch.Sketch, n map[sldprt.ConstraintKind]int, lines []*sketch.Line, g *sketch.GeometricConstraints) {
	for _, axis := range lines {
		if !axis.IsConstruction() {
			continue // the mirror axis is a construction line (a centerline)
		}
		for i := 0; i < len(lines); i++ {
			for j := i + 1; j < len(lines); j++ {
				if n[sldprt.Symmetric] <= 0 || sk.DegreesOfFreedom() <= 0 {
					return
				}
				a, b := lines[i], lines[j]
				if a == axis || b == axis || a.IsConstruction() || b.IsConstruction() {
					continue
				}
				if pa, pb := symmetricEndpoints(a, b, axis); pa != nil {
					g.AddSymmetry(pa[0], pb[0], axis)
					g.AddSymmetry(pa[1], pb[1], axis)
					n[sldprt.Symmetric]--
				}
			}
		}
	}
}

// symmetricEndpoints reports the corresponding endpoint pairs of two lines that are mirror images
// about axis: reflecting a's endpoints across the axis must land on b's endpoints. Returns the
// endpoints of a and the matching endpoints of b (index-aligned), or nil if the lines are not
// symmetric about the axis.
func symmetricEndpoints(a, b, axis *sketch.Line) (*[2]*sketch.Point, *[2]*sketch.Point) {
	ra := reflectAcrossLine(a.A.Position(), axis)
	rb := reflectAcrossLine(a.B.Position(), axis)
	switch {
	case coincides(ra, b.A.Position()) && coincides(rb, b.B.Position()):
		return &[2]*sketch.Point{a.A, a.B}, &[2]*sketch.Point{b.A, b.B}
	case coincides(ra, b.B.Position()) && coincides(rb, b.A.Position()):
		return &[2]*sketch.Point{a.A, a.B}, &[2]*sketch.Point{b.B, b.A}
	}
	return nil, nil
}

// reflectAcrossLine mirrors p across the infinite line through the axis segment.
func reflectAcrossLine(p m.Point2, axis *sketch.Line) m.Point2 {
	a := axis.A.Position()
	dir := axis.A.Position().VectorTo(axis.B.Position())
	dd := float64(dir.X*dir.X + dir.Y*dir.Y)
	if dd == 0 {
		return p
	}
	v := a.VectorTo(p)
	t := float64(v.X*dir.X+v.Y*dir.Y) / dd // projection parameter of p onto the axis
	foot := m.P2(a.X+m.Scalar(t)*dir.X, a.Y+m.Scalar(t)*dir.Y)
	return m.P2(2*foot.X-p.X, 2*foot.Y-p.Y)
}

// coincides reports whether two points are within geomEps.
func coincides(a, b m.Point2) bool {
	return math.Abs(float64(a.X-b.X)) < geomEps && math.Abs(float64(a.Y-b.Y)) < geomEps
}

// applyMidpoints binds each decoded midpoint relation to a standalone point that already sits at a
// line's midpoint, up to the decoded count and while free DOF remain. Because the point is exactly at
// the midpoint, the constraint pins it without moving it.
func applyMidpoints(sk *sketch.Sketch, n map[sldprt.ConstraintKind]int, points []*sketch.Point, lines []*sketch.Line, g *sketch.GeometricConstraints) {
	for _, p := range points {
		for _, l := range lines {
			if n[sldprt.Midpoint] <= 0 || sk.DegreesOfFreedom() <= 0 {
				return
			}
			mid := m.P2((l.A.Position().X+l.B.Position().X)/2, (l.A.Position().Y+l.B.Position().Y)/2)
			d := p.Position().DistanceTo(mid)
			if float64(d) < geomEps {
				g.AddMidpoint(p, l)
				n[sldprt.Midpoint]--
				break
			}
		}
	}
}

// applyToLines applies bind to each line satisfying it, up to the decoded count for kind and while
// the sketch keeps free DOF. A line is used once per kind (bind returns false to skip it).
func applyToLines(sk *sketch.Sketch, n map[sldprt.ConstraintKind]int, kind sldprt.ConstraintKind, lines []*sketch.Line, bind func(*sketch.Line) bool) {
	for _, l := range lines {
		if n[kind] <= 0 || sk.DegreesOfFreedom() <= 0 {
			return
		}
		if bind(l) {
			n[kind]--
		}
	}
}

// applyToLinePairs applies a two-line relation to each distinct pair that satisfies match, up to the
// decoded count and while DOF remains.
func applyToLinePairs[C any](sk *sketch.Sketch, n map[sldprt.ConstraintKind]int, kind sldprt.ConstraintKind, lines []*sketch.Line, match func(a, b *sketch.Line) bool, add func(a, b *sketch.Line) C) {
	for i := 0; i < len(lines); i++ {
		for j := i + 1; j < len(lines); j++ {
			if n[kind] <= 0 || sk.DegreesOfFreedom() <= 0 {
				return
			}
			if match(lines[i], lines[j]) {
				add(lines[i], lines[j])
				n[kind]--
			}
		}
	}
}

// applyToCurvePairs is applyToLinePairs for center+radius curves (concentric).
func applyToCurvePairs[C any](sk *sketch.Sketch, n map[sldprt.ConstraintKind]int, kind sldprt.ConstraintKind, cs []curve, match func(a, b curve) bool, add func(a, b sketch.CircularCurve) C) {
	for i := 0; i < len(cs); i++ {
		for j := i + 1; j < len(cs); j++ {
			if n[kind] <= 0 || sk.DegreesOfFreedom() <= 0 {
				return
			}
			if match(cs[i], cs[j]) {
				add(cs[i].h, cs[j].h)
				n[kind]--
			}
		}
	}
}

// applyTangents applies a line↔curve tangent to each pair whose perpendicular distance equals the
// curve radius, up to the decoded count and while DOF remains.
func applyTangents(sk *sketch.Sketch, n map[sldprt.ConstraintKind]int, lines []*sketch.Line, cs []curve, g *sketch.GeometricConstraints) {
	for _, l := range lines {
		for _, c := range cs {
			if n[sldprt.Tangent] <= 0 || sk.DegreesOfFreedom() <= 0 {
				return
			}
			if math.Abs(pointLineDistance(c.cx, c.cy, l)-c.r) < geomEps {
				g.AddTangent(l, c.h)
				n[sldprt.Tangent]--
			}
		}
	}
}

// curve is an emitted center+radius entity (arc or circle) with its geometry cached for matching.
type curve struct {
	cx, cy, r float64
	h         sketch.CircularCurve
}

func curvesOf(arcs []*sketch.Arc, circles []*sketch.Circle) []curve {
	out := make([]curve, 0, len(arcs)+len(circles))
	for _, a := range arcs {
		out = append(out, curve{a.Center.X, a.Center.Y, math.Hypot(a.Start.X-a.Center.X, a.Start.Y-a.Center.Y), a})
	}
	for _, c := range circles {
		out = append(out, curve{c.Center.X, c.Center.Y, float64(c.Radius), c})
	}
	return out
}

func lineHorizontal(l *sketch.Line) bool { return math.Abs(l.A.Y-l.B.Y) < geomEps }
func lineVertical(l *sketch.Line) bool   { return math.Abs(l.A.X-l.B.X) < geomEps }

// unitDir returns the normalised direction of a line, or (0,0) for a degenerate one.
func unitDir(l *sketch.Line) (float64, float64) {
	dx, dy := l.B.X-l.A.X, l.B.Y-l.A.Y
	n := math.Hypot(dx, dy)
	if n < geomEps {
		return 0, 0
	}
	return dx / n, dy / n
}

func perpendicular(a, b *sketch.Line) bool {
	ax, ay := unitDir(a)
	bx, by := unitDir(b)
	return math.Abs(ax*bx+ay*by) < geomEps
}

func parallel(a, b *sketch.Line) bool {
	ax, ay := unitDir(a)
	bx, by := unitDir(b)
	return math.Abs(ax*by-ay*bx) < geomEps
}

func collinear(a, b *sketch.Line) bool {
	return parallel(a, b) && math.Abs(pointLineDistance(b.A.X, b.A.Y, a)) < geomEps
}

func equalLength(a, b *sketch.Line) bool {
	la := math.Hypot(a.B.X-a.A.X, a.B.Y-a.A.Y)
	lb := math.Hypot(b.B.X-b.A.X, b.B.Y-b.A.Y)
	return math.Abs(la-lb) < geomEps
}

func concentric(a, b curve) bool {
	return math.Abs(a.cx-b.cx) < geomEps && math.Abs(a.cy-b.cy) < geomEps
}

// pointLineDistance is the perpendicular distance from (px,py) to the infinite line through l.
func pointLineDistance(px, py float64, l *sketch.Line) float64 {
	dx, dy := l.B.X-l.A.X, l.B.Y-l.A.Y
	n := math.Hypot(dx, dy)
	if n < geomEps {
		return math.Hypot(px-l.A.X, py-l.A.Y)
	}
	return math.Abs((px-l.A.X)*dy-(py-l.A.Y)*dx) / n
}
