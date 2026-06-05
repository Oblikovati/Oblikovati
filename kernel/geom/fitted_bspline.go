// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// Fitted (interpolating) B-splines pass *through* their input points, unlike the
// control-polygon constructors in bspline_curve*.go. The standard global-interpolation
// construction (The NURBS Book, A9.1) is used: chord-length parameters, averaged knots, and
// a basis-collocation linear system A·P = Q solved for the control points P. The curve is
// non-rational (unit weights) and cubic where there are enough points (degree min(3, n)).

// NewFittedBSplineCurve builds the non-rational B-spline that interpolates the given 3D
// points in order (contract: CreateFittedBSplineCurve). It needs >= 2 points and errors on
// coincident/degenerate input (no chord length to parameterize by).
func NewFittedBSplineCurve(points []math.Point3) (BSplineCurve, error) {
	degree, err := fitDegree(len(points))
	if err != nil {
		return BSplineCurve{}, err
	}
	ctrl, knots, err := fitInterpolation(coords3(points), degree)
	if err != nil {
		return BSplineCurve{}, err
	}
	return NewBSplineCurveUniformWeights(degree, points3(ctrl), knots)
}

// NewFittedBSplineCurve2d is the 2D analogue of [NewFittedBSplineCurve] (contract:
// CreateFittedBSplineCurve2d).
func NewFittedBSplineCurve2d(points []math.Point2) (BSplineCurve2d, error) {
	degree, err := fitDegree(len(points))
	if err != nil {
		return BSplineCurve2d{}, err
	}
	ctrl, knots, err := fitInterpolation(coords2(points), degree)
	if err != nil {
		return BSplineCurve2d{}, err
	}
	return NewBSplineCurve2dUniformWeights(degree, points2(ctrl), knots)
}

// fitDegree picks the interpolation degree for a point count: cubic when there are >= 4
// points, otherwise count−1 (degree 1 for two points, 2 for three). Errors below 2 points.
func fitDegree(count int) (int, error) {
	if count < 2 {
		return 0, fmt.Errorf("geom: fitted B-spline needs >= 2 points, got %d", count)
	}
	if count-1 < 3 {
		return count - 1, nil
	}
	return 3, nil
}

// fitInterpolation returns the control points (same coordinate shape as pts) and knot
// vector of the degree-p B-spline interpolating the rows of pts in order.
func fitInterpolation(pts [][]float64, p int) (ctrl [][]float64, knots []float64, err error) {
	ubar, err := chordParams(pts)
	if err != nil {
		return nil, nil, err
	}
	knots = averagedKnots(ubar, p)
	a := collocation(ubar, p, knots)
	b := cloneRows(pts)
	if err := gaussSolve(a, b); err != nil {
		return nil, nil, err
	}
	return b, knots, nil
}

// chordParams returns the chord-length parameters ū∈[0,1] for the points (ū₀=0, ūₙ=1),
// erroring when the total chord length is zero (all points coincident).
func chordParams(pts [][]float64) ([]float64, error) {
	n := len(pts)
	cum := make([]float64, n)
	for k := 1; k < n; k++ {
		cum[k] = cum[k-1] + coordDist(pts[k-1], pts[k])
	}
	total := cum[n-1]
	if total == 0 {
		return nil, fmt.Errorf("geom: fitted B-spline needs distinct points (total chord length 0)")
	}
	u := make([]float64, n)
	for k := range u {
		u[k] = cum[k] / total
	}
	u[n-1] = 1
	return u, nil
}

// averagedKnots returns the clamped knot vector (length n+p+2) from the parameters ū by
// the averaging rule (The NURBS Book eq. 9.8): p+1 zeros, then internal knots that are
// running averages of p consecutive parameters, then p+1 ones.
func averagedKnots(ubar []float64, p int) []float64 {
	n := len(ubar) - 1
	m := n + p + 1
	knots := make([]float64, m+1)
	for i := m - p; i <= m; i++ {
		knots[i] = 1
	}
	for j := 1; j <= n-p; j++ {
		sum := 0.0
		for i := j; i <= j+p-1; i++ {
			sum += ubar[i]
		}
		knots[j+p] = sum / float64(p)
	}
	return knots
}

// collocation builds the (n+1)×(n+1) basis matrix A where A[k][i] = Nᵢ,ₚ(ūₖ): the linear
// system whose solution is the interpolating control points.
func collocation(ubar []float64, p int, knots []float64) [][]float64 {
	n := len(ubar) - 1
	a := make([][]float64, n+1)
	for k := range a {
		row := make([]float64, n+1)
		span := findSpan(n, p, ubar[k], knots)
		basis := basisFuns(span, p, ubar[k], knots)
		for l := 0; l <= p; l++ {
			row[span-p+l] = basis[l]
		}
		a[k] = row
	}
	return a
}

// gaussSolve solves A·X = B in place (Gauss–Jordan with partial pivoting), leaving the
// solution in B. A is n×n; B is n×rhs. Errors when A is singular.
func gaussSolve(a, b [][]float64) error {
	for col := range a {
		piv := pivotRow(a, col)
		if stdmath.Abs(a[piv][col]) < 1e-12 {
			return fmt.Errorf("geom: interpolation matrix is singular at column %d", col)
		}
		a[col], a[piv] = a[piv], a[col]
		b[col], b[piv] = b[piv], b[col]
		eliminate(a, b, col)
	}
	for i := range a {
		scaleRow(b[i], 1/a[i][i])
	}
	return nil
}

// pivotRow returns the row at or below col with the largest magnitude in that column.
func pivotRow(a [][]float64, col int) int {
	piv := col
	for r := col + 1; r < len(a); r++ {
		if stdmath.Abs(a[r][col]) > stdmath.Abs(a[piv][col]) {
			piv = r
		}
	}
	return piv
}

// eliminate zeroes column col in every row but col, applying the same operations to b.
func eliminate(a, b [][]float64, col int) {
	for r := range a {
		if r == col {
			continue
		}
		f := a[r][col] / a[col][col]
		if f == 0 {
			continue
		}
		for c := col; c < len(a); c++ {
			a[r][c] -= f * a[col][c]
		}
		for c := range b[r] {
			b[r][c] -= f * b[col][c]
		}
	}
}

// scaleRow multiplies every entry of row by s.
func scaleRow(row []float64, s float64) {
	for c := range row {
		row[c] *= s
	}
}

// coordDist returns the Euclidean distance between two coordinate tuples of equal length.
func coordDist(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return stdmath.Sqrt(sum)
}

// cloneRows deep-copies a row-major matrix so it can be mutated in place.
func cloneRows(m [][]float64) [][]float64 {
	out := make([][]float64, len(m))
	for i, row := range m {
		out[i] = append([]float64(nil), row...)
	}
	return out
}

// coords3 / points3 convert between 3D points and {x,y,z} coordinate rows.
func coords3(pts []math.Point3) [][]float64 {
	out := make([][]float64, len(pts))
	for i, p := range pts {
		out[i] = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
	}
	return out
}

func points3(rows [][]float64) []math.Point3 {
	out := make([]math.Point3, len(rows))
	for i, r := range rows {
		out[i] = math.Point3{X: math.Scalar(r[0]), Y: math.Scalar(r[1]), Z: math.Scalar(r[2])}
	}
	return out
}

// coords2 / points2 convert between 2D points and {x,y} coordinate rows.
func coords2(pts []math.Point2) [][]float64 {
	out := make([][]float64, len(pts))
	for i, p := range pts {
		out[i] = []float64{float64(p.X), float64(p.Y)}
	}
	return out
}

func points2(rows [][]float64) []math.Point2 {
	out := make([]math.Point2, len(rows))
	for i, r := range rows {
		out[i] = math.P2(math.Scalar(r[0]), math.Scalar(r[1]))
	}
	return out
}
