// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// placement is one occurrence's rigid-body variable block during a solve: three position
// scalars plus a four-component quaternion (ADR-0011). The solver mutates these in place;
// matrix() rebuilds the placement transform from them on demand, and commit() writes the
// final transform back to the occurrence. A grounded occurrence exposes no variables — it
// is the anchor the rest of the assembly solves against (like a fixed sketch point).
type placement struct {
	occ      *occurrence.Occurrence
	pos      [3]math.Scalar // px, py, pz
	quat     [4]math.Scalar // qw, qx, qy, qz
	grounded bool
}

// newPlacement builds a placement warm-started from the occurrence's current transform,
// so a re-solve starts from the last-good configuration and converges in a few steps.
func newPlacement(o *occurrence.Occurrence) *placement {
	p := &placement{occ: o, grounded: o.Grounded()}
	p.warmStart()
	return p
}

// warmStart decomposes the occurrence's current placement into the variable block.
func (p *placement) warmStart() {
	m := p.occ.Transform()
	t := m.Translation()
	p.pos = [3]math.Scalar{t.X, t.Y, t.Z}
	q := math.QuaternionFromMatrix(m)
	p.quat = [4]math.Scalar{q.W, q.X, q.Y, q.Z}
}

// matrix rebuilds the placement transform from the current variables (translation after
// rotation). Quaternion.Matrix4 normalizes, so a slightly off-unit solver iterate still
// yields a valid rigid transform.
func (p *placement) matrix() math.Matrix4 {
	q := math.Quaternion{W: p.quat[0], X: p.quat[1], Y: p.quat[2], Z: p.quat[3]}
	return math.Translation4(math.V3(p.pos[0], p.pos[1], p.pos[2])).Mul(q.Matrix4())
}

// vars returns the solver variables for this placement — the seven scalar pointers, or
// nil when the occurrence is grounded (fixed).
func (p *placement) vars() []*math.Scalar {
	if p.grounded {
		return nil
	}
	return []*math.Scalar{&p.pos[0], &p.pos[1], &p.pos[2], &p.quat[0], &p.quat[1], &p.quat[2], &p.quat[3]}
}

// allVars returns all seven scalar pointers regardless of grounding, for per-occurrence
// DOF analysis (a grounded occurrence is reported as 0 DOF separately).
func (p *placement) allVars() []*math.Scalar {
	return []*math.Scalar{&p.pos[0], &p.pos[1], &p.pos[2], &p.quat[0], &p.quat[1], &p.quat[2], &p.quat[3]}
}

// commit writes the solved transform back to the occurrence (no-op when grounded).
func (p *placement) commit() {
	if p.grounded {
		return
	}
	p.occ.SetTransform(p.matrix())
}

// normalization is the residual keeping a free placement's quaternion unit length
// (Σq² − 1 = 0), pinning the gauge freedom of the four-component, three-DOF orientation.
type normalization struct{ p *placement }

// Residuals implements solve.Residual.
func (n normalization) Residuals() []float64 {
	q := n.p.quat
	return []float64{q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3] - 1}
}
