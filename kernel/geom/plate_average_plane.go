// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// PlateDomain is the flat 2D parameter domain Ω the Duchon plate solver (M6, P2+) builds
// over: the least-squares average plane through a degenerate corner's rail-constraint
// anchors. Rail points get PROJECTED into Ω (u,v), the plate energy is minimized there, and
// the result is LIFTED back to 3D (Origin + u·U + v·V + w·N). U, V, N form a right-handed
// orthonormal frame (U×V=N); this type carries only the frame, no plate math.
type PlateDomain struct {
	Origin  math.Point3
	U, V, N math.Vector3
}

// domainDegenerateTol is the relative floor on the in-plane spread ratio (minor/major, both
// in length units — analogous to conjugateDegenerateTol in elliptical_cylinder.go) below
// which the anchors carry no second independent in-plane direction: they are collinear (or
// worse), so no distinct plane normal is resolvable and AveragePlane must reject them rather
// than emit an arbitrary N. Dimensionless, so — unlike a coincidence length — it needs no
// model-relative (Resolution) scaling of its own.
const domainDegenerateTol = 1e-9

// AveragePlane fits the least-squares plane through anchors (contract: the 2D domain later
// plate-fill tasks solve over). Origin is the centroid; the anchors' covariance (scatter)
// matrix M = Σ(pᵢ−c)(pᵢ−c)ᵀ is diagonalized, N is the eigenvector of the SMALLEST eigenvalue
// (the direction of least spread — the plane normal), and U (the largest-eigenvalue
// eigenvector, re-orthogonalized against N), V = N×U complete a right-handed in-plane basis.
// Returns an error, carrying the measured spread and the expected minimum, when the anchors
// don't resolve two independent in-plane directions (see [domainDegenerateTol]) — e.g. fewer
// than 3 points, all anchors coincident within weld tolerance, or collinear anchors.
//
// Example:
//
//	dom, err := AveragePlane(cornerRailAnchors)
//	u, v := dom.Project(p) // p projected into Ω
func AveragePlane(anchors []math.Point3) (PlateDomain, error) {
	if len(anchors) < 3 {
		return PlateDomain{}, fmt.Errorf(
			"geom: AveragePlane needs >=3 anchors to fit a plane, got %d", len(anchors))
	}
	origin := centroidOf(anchors)
	values, vectors := jacobiEigen3(scatterMatrix(anchors, origin))
	n, u, v, err := planeFrameFromEigen(values, vectors, ResolutionForPoints(anchors))
	if err != nil {
		return PlateDomain{}, err
	}
	return PlateDomain{Origin: origin, U: u, V: v, N: n}, nil
}

// Project maps a 3D point into Ω's in-plane coordinates: (p−Origin)·U, (p−Origin)·V. The
// out-of-plane component is dropped; a caller that needs it reads (p−Origin)·N directly (see
// [PlateDomain.Lift]'s w parameter).
func (d PlateDomain) Project(p math.Point3) (u, v float64) {
	rel := d.Origin.VectorTo(p)
	return float64(rel.Dot(d.U)), float64(rel.Dot(d.V))
}

// Lift maps Ω coordinates (u,v) plus a signed height w along N back to 3D:
// Origin + u·U + v·V + w·N. w=0 stays exactly in the fitted plane; a nonzero w reconstructs
// a point that was off-plane.
//
// Example:
//
//	u, v := dom.Project(p)
//	w := dom.Origin.VectorTo(p).Dot(dom.N)
//	roundTrip := dom.Lift(u, v, w) // ≈ p
func (d PlateDomain) Lift(u, v, w float64) math.Point3 {
	offset := d.U.Scale(u).Add(d.V.Scale(v)).Add(d.N.Scale(w))
	return d.Origin.TranslateBy(offset)
}

// centroidOf returns the mean position of pts.
func centroidOf(pts []math.Point3) math.Point3 {
	var sum math.Vector3
	for _, p := range pts {
		sum = sum.Add(p.AsVector())
	}
	return sum.Scale(1 / float64(len(pts))).AsPoint()
}

// scatterMatrix returns M = Σ(pᵢ−c)(pᵢ−c)ᵀ, the unnormalized covariance whose eigenvectors
// are pts' principal axes (PCA plane fit) and whose eigenvalues are the squared spread along
// each axis.
func scatterMatrix(pts []math.Point3, c math.Point3) [3][3]float64 {
	var m [3][3]float64
	for _, p := range pts {
		d := c.VectorTo(p)
		row := [3]float64{float64(d.X), float64(d.Y), float64(d.Z)}
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				m[i][j] += row[i] * row[j]
			}
		}
	}
	return m
}

// planeFrameFromEigen picks N (smallest-eigenvalue eigenvector) and completes a right-handed
// orthonormal in-plane basis: U is the largest-eigenvalue eigenvector, re-orthogonalized
// against N to absorb jacobiEigen3's floating-point drift from exact orthogonality; V = N×U
// (so U×V=N — see the triple-product identity U×(N×U) = N(U·U) − U(U·N) = N since U⊥N,
// |U|=1). Errors when the anchors don't resolve two independent in-plane directions: either
// every eigenvalue is within weld tolerance of zero (all anchors coincide) or the two
// smallest are — collinear anchors, only the largest-eigenvalue axis carries spread.
func planeFrameFromEigen(values [3]float64, vectors [3][3]float64, res Resolution) (n, u, v math.Vector3, err error) {
	lo, mid, hi := sortEigenIndices(values)
	majorSpread := stdmath.Sqrt(stdmath.Max(0, values[hi]))
	minorSpread := stdmath.Sqrt(stdmath.Max(0, values[mid]))
	if majorSpread < res.Weld() {
		return math.Vector3{}, math.Vector3{}, math.Vector3{}, fmt.Errorf(
			"geom: AveragePlane anchors coincide within weld tolerance (largest spread %.6g <= weld %.6g); "+
				"need anchors spread over >=2 independent directions", majorSpread, res.Weld())
	}
	if minorSpread < domainDegenerateTol*majorSpread {
		return math.Vector3{}, math.Vector3{}, math.Vector3{}, fmt.Errorf(
			"geom: AveragePlane anchors are collinear/rank-deficient (in-plane spread ratio %.3e below floor "+
				"%.3e; spreads(lo,mid,hi)=(%.6g,%.6g,%.6g)); need >=2 independent in-plane directions to fit a plane",
			minorSpread/majorSpread, domainDegenerateTol, stdmath.Sqrt(stdmath.Max(0, values[lo])), minorSpread, majorSpread)
	}
	nUnit, err := math.UnitVector3FromVector(columnVector(vectors, lo))
	if err != nil {
		return math.Vector3{}, math.Vector3{}, math.Vector3{}, fmt.Errorf("geom: AveragePlane normal degenerate: %w", err)
	}
	n = nUnit.AsVector()
	u, err = orthogonalizeAgainst(columnVector(vectors, hi), n)
	if err != nil {
		return math.Vector3{}, math.Vector3{}, math.Vector3{}, err
	}
	return n, u, n.Cross(u), nil
}

// columnVector reads the k-th eigenvector (column k of jacobiEigen3's accumulated rotation
// matrix) out as a Vector3.
func columnVector(m [3][3]float64, k int) math.Vector3 {
	return math.V3(m[0][k], m[1][k], m[2][k])
}

// orthogonalizeAgainst returns u's component perpendicular to the unit vector n, renormalized
// — this absorbs jacobiEigen3's floating-point drift from exact orthogonality. Errors (rather
// than panicking) if the residual collapses to zero, which planeFrameFromEigen's caller-side
// majorSpread/minorSpread guards make practically unreachable, but this is an alpha kernel:
// never assume a numerical invariant holds without a checked path (project convention).
func orthogonalizeAgainst(u, n math.Vector3) (math.Vector3, error) {
	perp := u.Sub(n.Scale(u.Dot(n)))
	unit, err := math.UnitVector3FromVector(perp)
	if err != nil {
		return math.Vector3{}, fmt.Errorf("geom: AveragePlane in-plane axis degenerate after orthogonalizing against N: %w", err)
	}
	return unit.AsVector(), nil
}

// sortEigenIndices returns the indices of values in ascending order (lo, mid, hi) — a
// hand-rolled 3-element sorting network, cheaper and clearer than sort.Slice for a fixed
// triple.
func sortEigenIndices(values [3]float64) (lo, mid, hi int) {
	idx := [3]int{0, 1, 2}
	if values[idx[0]] > values[idx[1]] {
		idx[0], idx[1] = idx[1], idx[0]
	}
	if values[idx[1]] > values[idx[2]] {
		idx[1], idx[2] = idx[2], idx[1]
	}
	if values[idx[0]] > values[idx[1]] {
		idx[0], idx[1] = idx[1], idx[0]
	}
	return idx[0], idx[1], idx[2]
}

// jacobiMaxSweeps bounds the classical cyclic Jacobi iteration (each sweep rotates all 3
// off-diagonal pairs once). 3×3 symmetric eigendecomposition converges quadratically and
// is done in single digits of sweeps for any well-conditioned matrix; 50 is a generous cap
// so a pathological input degrades to "did not fully converge" rather than looping forever.
const jacobiMaxSweeps = 50

// jacobiConvergedFloor is the sum-of-squared-off-diagonals residual below which a itself is
// considered diagonal — a convergence/numeric guard (not a model length), so it is a bare
// constant by design (tol:numeric).
const jacobiConvergedFloor = 1e-28

// jacobiSkipFloor is the per-pivot "already zero" guard inside a single rotation — avoids a
// needless rotation (and its associated floating-point noise) on an off-diagonal entry
// that's already at machine precision (tol:numeric).
const jacobiSkipFloor = 1e-30

// jacobiEigen3 diagonalizes the symmetric 3×3 matrix a via the classical cyclic Jacobi
// rotation method (Golub & Van Loan, Matrix Computations §8.4): each sweep zeroes the three
// off-diagonal pairs in turn with an orthogonal (Givens) rotation, which converges
// quadratically for a symmetric matrix. Chosen over the codebase's existing analytic 3×3
// solver (kernel/ops.symmetricEigenvalues3, Smith 1961) because that one returns eigenvalues
// only: reconstructing eigenvectors from (A−λI) row cross-products degenerates exactly when
// two eigenvalues are close together — precisely the near-collinear-anchor case this file
// must DETECT (via [planeFrameFromEigen]'s domainDegenerateTol guard), not silently
// mishandle. Returns the (unsorted) eigenvalues and their unit eigenvectors as columns of
// vectors (vectors[i][k] is the i-th component of the k-th eigenvector).
func jacobiEigen3(a [3][3]float64) (values [3]float64, vectors [3][3]float64) {
	vectors = identity3()
	for sweep := 0; sweep < jacobiMaxSweeps; sweep++ {
		if jacobiSweep(&a, &vectors) < jacobiConvergedFloor {
			break
		}
	}
	return [3]float64{a[0][0], a[1][1], a[2][2]}, vectors
}

func identity3() [3][3]float64 {
	return [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
}

// jacobiSweep applies one cyclic sweep — rotations zeroing the (0,1),(0,2),(1,2)
// off-diagonal pairs in turn — and returns the resulting sum of squared off-diagonal
// entries (the convergence residual jacobiEigen3 checks against jacobiConvergedFloor).
func jacobiSweep(a *[3][3]float64, v *[3][3]float64) float64 {
	pairs := [3][2]int{{0, 1}, {0, 2}, {1, 2}}
	for _, pq := range pairs {
		jacobiRotate(a, v, pq[0], pq[1])
	}
	return a[0][1]*a[0][1] + a[0][2]*a[0][2] + a[1][2]*a[1][2]
}

// jacobiRotate applies the Givens rotation that zeroes a[p][q] using the t/c/s form (Golub &
// Van Loan eq. 8.4.1-2 / Numerical Recipes §11.1), which avoids the cancellation of computing
// tan θ directly when a[p][q] is already small. Updates a in place and accumulates the
// rotation into v's (p,q) columns.
func jacobiRotate(a, v *[3][3]float64, p, q int) {
	apq := a[p][q]
	if stdmath.Abs(apq) < jacobiSkipFloor {
		return
	}
	theta := (a[q][q] - a[p][p]) / (2 * apq)
	t := jacobiRotationTangent(theta)
	c := 1 / stdmath.Sqrt(t*t+1)
	s := t * c
	rotateMatrixEntries(a, p, q, c, s, t)
	rotateEigenvectorColumns(v, p, q, c, s)
}

// jacobiRotationTangent returns the smaller root (in magnitude) of t²+2tθ−1=0, the numerically
// stable choice (Numerical Recipes §11.1) that keeps the rotation angle in [−π/4, π/4].
func jacobiRotationTangent(theta float64) float64 {
	sign := 1.0
	if theta < 0 {
		sign = -1
	}
	return sign / (stdmath.Abs(theta) + stdmath.Sqrt(theta*theta+1))
}

// rotateMatrixEntries applies the Jacobi rotation's effect on a's diagonal, zeroes the (p,q)
// pair, and rotates the remaining row/column r ∉ {p,q} (Golub & Van Loan eq. 8.4.3-8.4.4).
func rotateMatrixEntries(a *[3][3]float64, p, q int, c, s, t float64) {
	apq := a[p][q]
	a[p][p] -= t * apq
	a[q][q] += t * apq
	a[p][q], a[q][p] = 0, 0
	r := thirdIndex(p, q)
	arp, arq := a[r][p], a[r][q]
	a[r][p], a[p][r] = c*arp-s*arq, c*arp-s*arq
	a[r][q], a[q][r] = s*arp+c*arq, s*arp+c*arq
}

// thirdIndex returns the index in {0,1,2} that is neither p nor q.
func thirdIndex(p, q int) int {
	return 3 - p - q
}

// rotateEigenvectorColumns accumulates the same (c,s) rotation into v's (p,q) columns, so v
// ends up holding the eigenvectors once a is diagonal.
func rotateEigenvectorColumns(v *[3][3]float64, p, q int, c, s float64) {
	for i := 0; i < 3; i++ {
		vip, viq := v[i][p], v[i][q]
		v[i][p] = c*vip - s*viq
		v[i][q] = s*vip + c*viq
	}
}
