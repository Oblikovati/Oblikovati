// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A stop face's own PARAMETER BOX, and the boundary edge a landing leaves it through.
//
// WHY THIS EXISTS. slideOntoWall lands a band's terminal station on the stop wall's IMPLICIT surface,
// which extends past the stop FACE's own trim (fillet_farend_trim.go says so in its own comment). On 14
// of the 29 corpus cases that run the far-end trim at least one station lands off the face
// (selfcross-trim-report.md §3.1) — complex/D8's r=30 band leaves its radius-24 corner round at that
// round's u=0 ruling and runs 6.064 of developed length onto the flat wall next door. Splitting the trim
// there needs one question answered exactly: IS this landing on the stop face, and if not, WHICH boundary
// edge did it leave through?
//
// WHY A BOX AND NOT A GENERAL CONTAINMENT. A ray-cast containment in the developed chart answers "inside?"
// but not "through which edge?", and it has to guess at a seam. Every stop face the far-end trim meets is
// a quadric wall bounded by RULINGS and RIMS — i.e. by iso-curves of its own surface — so the face IS a
// (u,v) box, the exit side IS a box bound, and the edge carrying that bound is found by the same test. A
// face whose boundary is not all-iso is DECLINED (the trim then keeps today's single-face behaviour), so
// this never guesses: it either recognises a box patch exactly or says nothing.

// paramBoxStations is how finely each boundary edge is sampled when the box is measured. An iso-curve is
// iso at every point, so 3 samples would decide it; 8 spans leaves margin for a rim carried as a spline
// fit whose parameterisation wanders slightly off its analytic iso value.
const paramBoxStations = 8

// paramBox is a face's own extent in its surface's (u,v) chart, with periodic parameters UNWRAPPED — so a
// cylinder wall spanning u ∈ [−π/2, 0] reads as that, not as the [0, 2π) branch ParamAt returns.
type paramBox struct{ uLo, uHi, vLo, vHi float64 }

// boxSide names one bound of a parameter box (which is also one boundary edge of the face).
type boxSide int

const (
	sideInside boxSide = iota // not outside any bound
	sideULo
	sideUHi
	sideVLo
	sideVHi
)

// faceParamBox measures f's outer boundary's parameter box; see ringParamBox for the contract.
func faceParamBox(f *topo.Face, tol float64) (paramBox, bool) {
	if f.Geometry() == nil {
		return paramBox{}, false
	}
	return ringParamBox(f.Geometry(), originalHostSegs(f), tol)
}

// ringParamBox measures a boundary ring's parameter box on its own surface, or declines when the ring is
// not a box patch — when any edge varies in BOTH parameters, or when the ring wraps a periodic seam
// (where the tessellate.Unwrap is not a polygon at all). tol is a model-relative LENGTH: the box is measured in the
// surface's metric chart, so an angular parameter is compared after arc-length scaling and no bare
// epsilon is used (ADR-0042). It is topo-free, the same seam chainRetrimLoop keeps.
func ringParamBox(s geom.Surface, segs []endSeg, tol float64) (paramBox, bool) {
	edges, ok := ringEdgeParams(s, segs)
	if !ok {
		return paramBox{}, false
	}
	su, sv := tessellate.MetricScale(s)
	for _, e := range edges {
		if spanOf(e.us)*su > tol && spanOf(e.vs)*sv > tol {
			return paramBox{}, false // a boundary edge that is not an iso-curve: this ring is not a box
		}
	}
	return boxOfEdges(edges), true
}

// edgeParams is one boundary edge of a face, developed into the face's own (u,v) chart.
type edgeParams struct{ us, vs []float64 }

// ringEdgeParams develops every segment of a face's outer ring into the surface's (u,v) chart, sampling
// each segment through its OWN carried curve and unwrapping the periodic parameter along the ring so the
// samples read continuously. It declines a ring shorter than a triangle or one whose tessellate.Unwrap fails.
func ringEdgeParams(s geom.Surface, segs []endSeg) ([]edgeParams, bool) {
	if len(segs) < 3 {
		return nil, false
	}
	us, vs := ringParamSequence(s, segs)
	var ok bool
	if tessellate.IsPeriodic(s.UDomain()) {
		if us, ok = tessellate.Unwrap(us); !ok {
			return nil, false
		}
	}
	if tessellate.IsPeriodic(s.VDomain()) {
		if vs, ok = tessellate.Unwrap(vs); !ok {
			return nil, false
		}
	}
	return sliceIntoEdges(us, vs, len(segs)), true
}

// ringParamSequence is the ring's samples inverted to (u,v), in traversal order, paramBoxStations+1 per
// segment (the segments' shared endpoints are sampled twice, which the tessellate.Unwrap needs and the box ignores).
func ringParamSequence(s geom.Surface, segs []endSeg) (us, vs []float64) {
	for _, seg := range segs {
		for i := 0; i <= paramBoxStations; i++ {
			u, v := s.ParamAt(segPointAt(seg, float64(i)/float64(paramBoxStations)))
			us, vs = append(us, u), append(vs, v)
		}
	}
	return us, vs
}

// sliceIntoEdges cuts the flat ring sample sequence back into per-edge blocks.
func sliceIntoEdges(us, vs []float64, n int) []edgeParams {
	k := paramBoxStations + 1
	out := make([]edgeParams, n)
	for i := range n {
		out[i] = edgeParams{us: us[i*k : (i+1)*k], vs: vs[i*k : (i+1)*k]}
	}
	return out
}

// boxOfEdges is the bounding box of every developed ring sample.
func boxOfEdges(edges []edgeParams) paramBox {
	b := paramBox{stdmath.Inf(1), stdmath.Inf(-1), stdmath.Inf(1), stdmath.Inf(-1)}
	for _, e := range edges {
		b.uLo, b.uHi = stdmath.Min(b.uLo, minOfFloats(e.us)), stdmath.Max(b.uHi, maxOfFloats(e.us))
		b.vLo, b.vHi = stdmath.Min(b.vLo, minOfFloats(e.vs)), stdmath.Max(b.vHi, maxOfFloats(e.vs))
	}
	return b
}

// boxParamAt inverts p onto s and places a periodic parameter on the BOX's own branch (the nearest
// multiple of 2π to the box centre), so a landing just past a face whose chart straddles ParamAt's 0/2π
// cut reads as "just past" rather than as a whole period away.
func boxParamAt(s geom.Surface, b paramBox, p math.Point3) (u, v float64) {
	u, v = s.ParamAt(p)
	if tessellate.IsPeriodic(s.UDomain()) {
		u = onBranch(u, (b.uLo+b.uHi)/2)
	}
	if tessellate.IsPeriodic(s.VDomain()) {
		v = onBranch(v, (b.vLo+b.vHi)/2)
	}
	return u, v
}

// onBranch shifts a by whole periods to the branch nearest ref.
func onBranch(a, ref float64) float64 {
	return a + 2*stdmath.Pi*stdmath.Round((ref-a)/(2*stdmath.Pi))
}

// boxSideOfPoint is boxExitSide for a 3D point: invert it onto the surface on the box's own branch, then
// name the bound it lies outside of (sideInside when it is on the face's own patch).
func boxSideOfPoint(s geom.Surface, b paramBox, p math.Point3, tol float64) boxSide {
	u, v := boxParamAt(s, b, p)
	return boxExitSide(s, b, u, v, tol)
}

// boxExitSide reports which bound of b the parameter point (u,v) lies outside of, by the LARGEST metric
// excursion; sideInside when it is within tol of every bound (a landing exactly ON a bound — which is
// where a correct trim ends — counts as inside).
func boxExitSide(s geom.Surface, b paramBox, u, v, tol float64) boxSide {
	su, sv := tessellate.MetricScale(s)
	best, side := tol, sideInside
	for _, c := range []struct {
		d    float64
		side boxSide
	}{{(b.uLo - u) * su, sideULo}, {(u - b.uHi) * su, sideUHi}, {(b.vLo - v) * sv, sideVLo}, {(v - b.vHi) * sv, sideVHi}} {
		if c.d > best {
			best, side = c.d, c.side
		}
	}
	return side
}

// boxSideBound is the box's own parameter value on the named side.
func boxSideBound(b paramBox, side boxSide) float64 {
	switch side {
	case sideULo:
		return b.uLo
	case sideUHi:
		return b.uHi
	case sideVLo:
		return b.vLo
	default:
		return b.vHi
	}
}

// boxSideParam is the coordinate the named side bounds (u for a u-side, v for a v-side).
func boxSideParam(side boxSide, u, v float64) float64 {
	if side == sideULo || side == sideUHi {
		return u
	}
	return v
}

// faceNeighbourOnBoxSide returns the face across the boundary EDGE of f that carries the named box side —
// the iso-curve every one of whose samples sits on that bound. It declines when no edge or more than one
// edge carries the bound (an ambiguous patch), so the split never picks a neighbour by proximity.
func faceNeighbourOnBoxSide(f *topo.Face, b paramBox, side boxSide, tol float64) (*topo.Face, bool) {
	i, ok := ringIndexOnBoxSide(f.Geometry(), originalHostSegs(f), b, side, tol)
	uses := boundaryUses(f)
	if !ok || i >= len(uses) {
		return nil, false
	}
	nb := otherFace(uses[i].Edge(), f)
	return nb, nb != nil
}

// ringIndexOnBoxSide is the index of the ONE ring segment lying wholly on the box side's bound — the
// topo-free core of faceNeighbourOnBoxSide. It declines when no segment or more than one carries the
// bound, so a neighbour is never picked by proximity.
func ringIndexOnBoxSide(s geom.Surface, segs []endSeg, b paramBox, side boxSide, tol float64) (int, bool) {
	edges, ok := ringEdgeParams(s, segs)
	if !ok {
		return 0, false
	}
	su, sv := tessellate.MetricScale(s)
	scale := su
	if side == sideVLo || side == sideVHi {
		scale = sv
	}
	hit := -1
	for i, e := range edges {
		if !edgeOnBound(e, side, boxSideBound(b, side), scale, tol) {
			continue
		}
		if hit >= 0 {
			return 0, false // two segments on one bound: ambiguous, decline
		}
		hit = i
	}
	return hit, hit >= 0
}

// edgeOnBound reports whether every sample of one ring edge sits on the named bound within tol (metric).
func edgeOnBound(e edgeParams, side boxSide, bound, scale, tol float64) bool {
	vals := e.us
	if side == sideVLo || side == sideVHi {
		vals = e.vs
	}
	for _, x := range vals {
		if stdmath.Abs(x-bound)*scale > tol {
			return false
		}
	}
	return true
}

// boundaryUses returns the outer loop's edge uses in traversal order (the same order originalHostSegs
// walks, so index i of one indexes the other).
func boundaryUses(f *topo.Face) []*topo.EdgeUse {
	l := outerHostLoop(f)
	if l == nil {
		return nil
	}
	return l.EdgeUses()
}

// spanOf is max−min of a sample run.
func spanOf(a []float64) float64 { return maxOfFloats(a) - minOfFloats(a) }

func minOfFloats(a []float64) float64 {
	m := stdmath.Inf(1)
	for _, x := range a {
		m = stdmath.Min(m, x)
	}
	return m
}

func maxOfFloats(a []float64) float64 {
	m := stdmath.Inf(-1)
	for _, x := range a {
		m = stdmath.Max(m, x)
	}
	return m
}
