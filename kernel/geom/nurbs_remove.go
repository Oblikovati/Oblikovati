// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"
)

// Knot removal on homogeneous control points (Tiller's algorithm, Piegl & Tiller
// A5.8). Unlike insertion, removal is *approximate*: a knot can be removed only if
// doing so keeps the curve within a tolerance of the original. Removal is half of a
// rebuild — insert a dense knot set, then remove what the shape does not need to
// reach a clean, minimal control polygon (M36-F01). The tolerance is measured on the
// homogeneous control points (equal to model space for a non-rational curve).

// removeKnotHomog tries to remove the knot u (last index r, multiplicity s) up to num
// times from the homogeneous curve, succeeding only while each candidate control point
// stays within tol. It returns the new knots, control points and the removal count.
func removeKnotHomog(p int, knots []float64, pw []hpoint4, u float64, r, s, num int, tol float64) (newU []float64, newPw []hpoint4, removed int) {
	work := append([]hpoint4(nil), pw...)
	n := len(pw) - 1
	ord := p + 1
	first, last := r-p, r-s
	temp := make([]hpoint4, 2*p+2)
	t := 0
	for ; t < num; t++ {
		if !removeOnce(ord, knots, work, temp, u, first, last, t, tol) {
			break
		}
		first--
		last++
	}
	if t == 0 {
		return append([]float64(nil), knots...), work, 0
	}
	return shiftRemovedKnots(knots, r, t), shiftRemovedCtrl(work, p, r, s, t, n), t
}

// removeOnce computes the candidate control points for removal pass t in temp, tests
// removability against tol, and on success writes them back into work. It returns
// whether this pass removed the knot.
func removeOnce(ord int, knots []float64, work, temp []hpoint4, u float64, first, last, t int, tol float64) bool {
	off := first - 1
	temp[0] = work[off]
	temp[last+1-off] = work[last+1]
	i, j := first, last
	ii, jj := 1, last-off
	for j-i > t {
		alfi := (u - knots[i]) / (knots[i+ord+t] - knots[i])
		alfj := (u - knots[j-t]) / (knots[j+ord] - knots[j-t])
		temp[ii] = work[i].sub(temp[ii-1].scale(1 - alfi)).scale(1 / alfi)
		temp[jj] = work[j].sub(temp[jj+1].scale(alfj)).scale(1 / (1 - alfj))
		i, ii, j, jj = i+1, ii+1, j-1, jj-1
	}
	if !removable(ord, knots, work, temp, u, i, j, ii, jj, t, tol) {
		return false
	}
	writeBackRemoval(work, temp, off, first, last, t)
	return true
}

// removable applies A5.8's two-case deviation test: the symmetric (j−i<t) case compares
// the two converging temps, the asymmetric case reconstructs the original control point.
func removable(ord int, knots []float64, work, temp []hpoint4, u float64, i, j, ii, jj, t int, tol float64) bool {
	if j-i < t {
		return temp[ii-1].dist(temp[jj+1]) <= tol
	}
	alfi := (u - knots[i]) / (knots[i+ord+t] - knots[i])
	candidate := temp[ii+t+1].scale(alfi).add(temp[ii-1].scale(1 - alfi))
	return work[i].dist(candidate) <= tol
}

// writeBackRemoval copies the accepted candidate control points from temp into work.
func writeBackRemoval(work, temp []hpoint4, off, first, last, t int) {
	i, j := first, last
	for j-i > t {
		work[i] = temp[i-off]
		work[j] = temp[j-off]
		i, j = i+1, j-1
	}
}

// shiftRemovedKnots drops t copies of the removed knot from knots (the knots above index r
// slide down by t), returning the shortened knot vector.
func shiftRemovedKnots(knots []float64, r, t int) []float64 {
	m := len(knots) - 1
	out := append([]float64(nil), knots...)
	for k := r + 1; k <= m; k++ {
		out[k-t] = out[k]
	}
	return out[:len(knots)-t]
}

// shiftRemovedCtrl closes the t-wide gap A5.8 opens around the removed knot, returning
// the shortened control-point slice.
func shiftRemovedCtrl(work []hpoint4, p, r, s, t, n int) []hpoint4 {
	fout := (2*r - s - p) / 2
	i, j := fout, fout
	for k := 1; k < t; k++ {
		if k%2 == 1 {
			i++
		} else {
			j--
		}
	}
	for k := i + 1; k <= n; k++ {
		work[j] = work[k]
		j++
	}
	return work[:len(work)-t]
}

// interiorKnot validates a removal request and resolves u's last index r, multiplicity
// s, and the effective removal count (capped at s — a knot cannot be removed more times
// than it is present). It errors when num < 1 or u is not an interior knot.
func interiorKnot(p int, knots []float64, u float64, num int) (r, s, effNum int, err error) {
	if num < 1 {
		return 0, 0, 0, fmt.Errorf("geom: knot removal count %d must be >= 1", num)
	}
	lo, hi := knots[p], knots[len(knots)-1-p]
	if u <= lo+knotEps || u >= hi-knotEps {
		return 0, 0, 0, fmt.Errorf("geom: knot %g is not interior to (%g, %g); only interior knots are removable", u, lo, hi)
	}
	r, s = lastKnotIndex(knots, u)
	if s == 0 {
		return 0, 0, 0, fmt.Errorf("geom: knot %g does not appear in the knot vector", u)
	}
	return r, s, min(num, s), nil
}

// lastKnotIndex returns the highest index at which u appears in knots (within knotEps) and
// u's multiplicity; mult is 0 when u is absent.
func lastKnotIndex(knots []float64, u float64) (idx, mult int) {
	for i, k := range knots {
		if stdmath.Abs(k-u) <= knotEps {
			idx = i
			mult++
		}
	}
	return idx, mult
}
