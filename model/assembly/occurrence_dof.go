// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// Per-occurrence DOF split (#1980): the scalar free-DOF count is decomposed into translational and
// rotational parts, with the DOF centre and the free translation/rotation axis directions, so a
// "show degrees of freedom" glyph can be drawn. The split comes from a twist-space mobility analysis:
// the residuals' sensitivity to the six canonical rigid twists (three translations, three rotations
// about the occurrence origin) forms a 6-column Jacobian whose null space is the free motion.

const dofProbeEps = 1e-6

// DOFSplit is an occurrence's remaining freedom split into translational and rotational counts, with
// the DOF centre and the free translation/rotation axis directions (#1980).
type DOFSplit struct {
	TranslationCount int
	RotationCount    int
	Center           math.Point3
	TranslationAxes  []math.Vector3
	RotationAxes     []math.Vector3
}

// occurrenceDOFSplit decomposes an occurrence's free motion. It keeps the existing scalar DOF as the
// authoritative total (so translation+rotation always sums to it) and classifies the freedom via the
// twist Jacobian's null space (#1980).
func occurrenceDOFSplit(p *placement, residuals []solve.Residual, total int) DOFSplit {
	origin := math.P3(p.pos[0], p.pos[1], p.pos[2])
	split := DOFSplit{Center: origin}
	if p.grounded || total == 0 {
		return split
	}
	j := twistJacobian(p, residuals)
	transAxes := nullSpaceBasis(columnRange(j, 0, 3)) // free pure-translation directions
	split.TranslationCount = len(transAxes)
	split.TranslationAxes = unitVectors(transAxes, 0)
	split.RotationCount = total - split.TranslationCount
	if split.RotationCount > 0 {
		axes, center := rotationDOF(j, origin, split.RotationCount)
		split.RotationAxes, split.Center = axes, center
	}
	return split
}

// rotationDOF extracts up to count independent rotation-axis directions and the DOF centre (a point
// on the first rotational screw axis) from the full twist null space.
func rotationDOF(j [][]float64, origin math.Point3, count int) ([]math.Vector3, math.Point3) {
	var axes []math.Vector3
	center := origin
	for _, twist := range nullSpaceBasis(j) {
		v := math.V3(math.Scalar(twist[0]), math.Scalar(twist[1]), math.Scalar(twist[2]))
		w := math.V3(math.Scalar(twist[3]), math.Scalar(twist[4]), math.Scalar(twist[5]))
		unit, err := math.UnitVector3FromVector(w)
		if err != nil || directionSeen(axes, unit.AsVector()) {
			continue
		}
		if len(axes) == 0 {
			center = screwCenter(origin, v, w)
		}
		axes = append(axes, unit.AsVector())
		if len(axes) == count {
			break
		}
	}
	return axes, center
}

// screwCenter returns the point on a twist's screw axis nearest the origin: origin + (ω×v)/|ω|². For
// a rotation about an axis through the origin (v=0) it is the origin; for an offset axis it recovers
// a point on that axis.
func screwCenter(origin math.Point3, v, w math.Vector3) math.Point3 {
	l2 := float64(w.Dot(w))
	if l2 < 1e-12 {
		return origin
	}
	arm := w.Cross(v).Scale(math.Scalar(1 / l2))
	return origin.TranslateBy(arm)
}

// twistJacobian builds the residuals' m×6 sensitivity to the six canonical twists (three
// translations, three rotations about the occurrence origin) by one-sided finite differences.
func twistJacobian(p *placement, residuals []solve.Residual) [][]float64 {
	base := evalResidualValues(residuals)
	cols := make([][]float64, 6)
	for c := range 6 {
		cols[c] = twistColumn(p, residuals, base, c)
	}
	rows := make([][]float64, len(base))
	for i := range rows {
		rows[i] = []float64{cols[0][i], cols[1][i], cols[2][i], cols[3][i], cols[4][i], cols[5][i]}
	}
	return rows
}

// twistColumn perturbs the occurrence by canonical twist c, re-evaluates the residuals, restores the
// placement, and returns the per-unit sensitivity column.
func twistColumn(p *placement, residuals []solve.Residual, base []float64, c int) []float64 {
	restore := applyTwist(p, c)
	after := evalResidualValues(residuals)
	restore()
	col := make([]float64, len(base))
	for i := range col {
		col[i] = (after[i] - base[i]) / dofProbeEps
	}
	return col
}

// applyTwist perturbs the placement by canonical twist c (0..2 translate along an axis, 3..5 rotate
// about the occurrence origin) and returns a function that restores the original placement.
func applyTwist(p *placement, c int) func() {
	if c < 3 {
		old := p.pos[c]
		p.pos[c] += dofProbeEps
		return func() { p.pos[c] = old }
	}
	old := p.quat
	axis, _ := math.NewUnitVector3(boolAxis(c-3, 0), boolAxis(c-3, 1), boolAxis(c-3, 2))
	dq := math.QuaternionFromAxisAngle(axis, dofProbeEps)
	q := dq.Mul(math.Quaternion{W: p.quat[0], X: p.quat[1], Y: p.quat[2], Z: p.quat[3]})
	p.quat = [4]math.Scalar{q.W, q.X, q.Y, q.Z}
	return func() { p.quat = old }
}

// boolAxis returns 1 when component k is axis i, else 0 — the i-th canonical unit vector.
func boolAxis(i, k int) math.Scalar {
	if i == k {
		return 1
	}
	return 0
}

// evalResidualValues concatenates every residual's current values.
func evalResidualValues(residuals []solve.Residual) []float64 {
	var out []float64
	for _, r := range residuals {
		out = append(out, r.Residuals()...)
	}
	return out
}

// unitVectors normalises each null-space basis vector's first three components (a translation
// direction), skipping degenerate ones; component offset selects translation (0) or rotation (3).
func unitVectors(basis [][]float64, offset int) []math.Vector3 {
	var out []math.Vector3
	for _, b := range basis {
		v := math.V3(math.Scalar(b[offset]), math.Scalar(b[offset+1]), math.Scalar(b[offset+2]))
		if unit, err := math.UnitVector3FromVector(v); err == nil {
			out = append(out, unit.AsVector())
		}
	}
	return out
}

// directionSeen reports whether dir (up to sign) is already among the collected axes.
func directionSeen(axes []math.Vector3, dir math.Vector3) bool {
	for _, a := range axes {
		if stdmath.Abs(float64(a.Dot(dir))) > 0.999 {
			return true
		}
	}
	return false
}
