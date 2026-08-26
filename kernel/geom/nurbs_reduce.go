// SPDX-License-Identifier: GPL-2.0-only

package geom

// Degree reduction on homogeneous control points — the inverse of degree elevation and,
// like knot removal, *approximate*: a curve can drop a degree only if its shape admits a
// lower-degree representation within a tolerance (M36-F01). Rather than A5.11's in-place
// error bookkeeping, this uses the equivalent decompose→reduce→recompose route, which
// reuses the already-tested knot insertion (Bézier extraction) and knot removal (cleanup):
//
//  1. insert knots until the curve is a sequence of Bézier segments;
//  2. lower each segment to degree p−1 (Piegl & Tiller eq. 5.40), rejecting the whole
//     curve if any segment's deviation — measured by RE-ELEVATING the reduced segment and
//     comparing to the original, an exact error rather than an estimate — exceeds tol;
//  3. re-join the reduced segments into a C^0 degree-(p−1) B-spline.
//
// The caller then removes the now-redundant interior knots (curves directly, surfaces via
// the all-columns-agree RemoveKnot), so a surface keeps one consistent knot vector.

// degreeReduceHomog reduces the homogeneous B-spline (degree p) by one degree to its C^0
// Bézier-joined form within tol, returning ok=false when any segment is not reducible.
// The result still carries the redundant join knots; the caller cleans them up.
func degreeReduceHomog(p int, knots []float64, pw []hpoint4, tol float64) (newU []float64, newPw []hpoint4, ok bool) {
	if p < 2 {
		return nil, nil, false
	}
	breaks := distinctValues(knots)
	segs := decomposeBezier(p, knots, pw, breaks)
	reduced := make([][]hpoint4, len(segs))
	for i, seg := range segs {
		r, segOK := reduceBezierSegment(seg, tol)
		if !segOK {
			return nil, nil, false
		}
		reduced[i] = r
	}
	newU, newPw = recomposeBezier(p-1, breaks, reduced)
	return newU, newPw, true
}

// distinctValues returns the distinct knot values of knots in order (consecutive equal
// entries collapse to one), i.e. the Bézier break points including the two clamped ends.
func distinctValues(knots []float64) []float64 {
	var out []float64
	for i, k := range knots {
		if i == 0 || k != knots[i-1] {
			out = append(out, k)
		}
	}
	return out
}

// decomposeBezier inserts every interior break up to multiplicity p so the curve becomes
// a chain of Bézier segments, then slices out each segment's p+1 control points (segments
// overlap by one shared endpoint).
func decomposeBezier(p int, knots []float64, pw []hpoint4, breaks []float64) [][]hpoint4 {
	work := pw
	for _, v := range breaks[1 : len(breaks)-1] {
		if m := knotMultiplicity(knots, v); m < p {
			knots, work = insertKnotHomog(p, knots, work, v, p-m)
		}
	}
	nseg := len(breaks) - 1
	segs := make([][]hpoint4, nseg)
	for s := range nseg {
		segs[s] = append([]hpoint4(nil), work[s*p:s*p+p+1]...)
	}
	return segs
}

// reduceBezierSegment lowers one Bézier segment from degree p to p−1 and accepts it only
// when re-elevating the result stays within tol of the original at every control point.
func reduceBezierSegment(seg []hpoint4, tol float64) (reduced []hpoint4, ok bool) {
	p := len(seg) - 1
	q := bezierReduceCtrl(seg, p)
	if bezierReduceError(seg, q, p) > tol {
		return nil, false
	}
	return q, true
}

// bezierReduceCtrl computes the degree-(p−1) control points by meeting the left-to-right
// and right-to-left reduction recurrences (eq. 5.40) in the middle, which is exact for a
// genuinely reducible segment and well-conditioned otherwise.
func bezierReduceCtrl(seg []hpoint4, p int) []hpoint4 {
	qf := reduceForward(seg, p)
	qb := reduceBackward(seg, p)
	r := (p - 1) / 2
	q := make([]hpoint4, p)
	for i := range p {
		if i <= r {
			q[i] = qf[i]
		} else {
			q[i] = qb[i]
		}
	}
	return q
}

// reduceForward solves the elevation relation P_i = (i/p)·Q_{i−1} + (1−i/p)·Q_i for Q from
// the left (Q_0 = P_0); accurate for the left half of the segment.
func reduceForward(seg []hpoint4, p int) []hpoint4 {
	q := make([]hpoint4, p)
	q[0] = seg[0]
	for i := 1; i < p; i++ {
		a := float64(i) / float64(p)
		q[i] = seg[i].sub(q[i-1].scale(a)).scale(1 / (1 - a))
	}
	return q
}

// reduceBackward solves the same relation from the right (Q_{p−1} = P_p); accurate for the
// right half of the segment.
func reduceBackward(seg []hpoint4, p int) []hpoint4 {
	q := make([]hpoint4, p)
	q[p-1] = seg[p]
	for i := p - 2; i >= 0; i-- {
		a := float64(i+1) / float64(p)
		q[i] = seg[i+1].sub(q[i+1].scale(1 - a)).scale(1 / a)
	}
	return q
}

// bezierReduceError returns the largest control-point deviation between the original
// segment and the degree-(p−1) candidate re-elevated to degree p — an exact reducibility
// measure (zero exactly when the segment is one degree reducible).
func bezierReduceError(seg, q []hpoint4, p int) float64 {
	elevated := elevateBezier1(q, p-1)
	maxErr := 0.0
	for i := 0; i <= p; i++ {
		if d := seg[i].dist(elevated[i]); d > maxErr {
			maxErr = d
		}
	}
	return maxErr
}

// elevateBezier1 raises a single Bézier polygon from degree d to d+1 in closed form:
// P̃_i = (i/(d+1))·Q_{i−1} + (1 − i/(d+1))·Q_i, preserving the end points.
func elevateBezier1(q []hpoint4, d int) []hpoint4 {
	out := make([]hpoint4, d+2)
	out[0] = q[0]
	out[d+1] = q[d]
	for i := 1; i <= d; i++ {
		a := float64(i) / float64(d+1)
		out[i] = q[i-1].scale(a).add(q[i].scale(1 - a))
	}
	return out
}

// recomposeBezier joins the reduced (degree pr) Bézier segments into one C^0 B-spline:
// segments share endpoints and every interior break carries multiplicity pr.
func recomposeBezier(pr int, breaks []float64, reduced [][]hpoint4) (knots []float64, pw []hpoint4) {
	pw = append([]hpoint4(nil), reduced[0]...)
	for s := 1; s < len(reduced); s++ {
		pw = append(pw, reduced[s][1:]...)
	}
	return bezierJoinKnots(pr, breaks), pw
}

// bezierJoinKnots builds the clamped knot vector of a C^0 chain of degree-pr Béziers over
// the given break points (interior breaks at multiplicity pr, ends at pr+1).
func bezierJoinKnots(pr int, breaks []float64) []float64 {
	nseg := len(breaks) - 1
	knots := make([]float64, 0, 2*(pr+1)+(nseg-1)*pr)
	for k := 0; k <= pr; k++ {
		knots = append(knots, breaks[0])
	}
	for s := 1; s < nseg; s++ {
		for range pr {
			knots = append(knots, breaks[s])
		}
	}
	for k := 0; k <= pr; k++ {
		knots = append(knots, breaks[nseg])
	}
	return knots
}
