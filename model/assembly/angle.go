// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// AngleConstraint holds a fixed angle between two directions (plane normals or axes). The
// undirected solution's single residual is cos θ − cos(target), smooth and singularity-free but
// unsigned. The directed and reference-vector solutions measure a SIGNED angle about a reference
// axis, so the angle can be held negative or past 180° (#1972): directed signs about an implied
// axis captured from the reference configuration; reference-vector signs about an explicit third
// entity.
type AngleConstraint struct {
	*constraintBase
	angle      float64
	solution   types.AngleConstraintSolutionType
	ref        *anchor      // reference-vector solution: the explicit axis to measure about (nil otherwise)
	implied    math.Vector3 // directed solution: the implied axis, captured at the first solve evaluation
	impliedSet bool
}

// Value returns the target angle (radians).
func (c *AngleConstraint) Value() float64 { return c.angle }

// SetValue overrides the constrained angle (a positional representation, M12-F04).
func (c *AngleConstraint) SetValue(v float64) { c.angle = v }

// SolutionType returns how the angle is measured (undirected/directed/reference-vector).
func (c *AngleConstraint) SolutionType() types.AngleConstraintSolutionType { return c.solution }

// anchors returns the constraint inputs; the reference-vector axis is a third anchor (on its own
// occurrence) so the per-occurrence view sees it too.
func (c *AngleConstraint) anchors() []anchor {
	if c.ref != nil {
		return []anchor{c.a, c.b, *c.ref}
	}
	return []anchor{c.a, c.b}
}

// bind returns the angle's residual source, branching on the solution type.
func (c *AngleConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		dA, dB := worldDir(pa.matrix(), c.a.prim), worldDir(pb.matrix(), c.b.prim)
		switch c.solution {
		case types.AngleSolutionReferenceVector:
			axis := worldDir(b(c.ref.occ).matrix(), c.ref.prim)
			return signedAngleResiduals(dA, dB, axis, c.angle)
		case types.AngleSolutionDirected:
			return c.directedResiduals(dA, dB)
		default:
			return undirectedAngleResiduals(dA, dB, c.angle)
		}
	})
}

// undirectedAngleResiduals returns the cosine residual between the two directions — unsigned and
// singularity-free, the default (and unchanged) behavior.
func undirectedAngleResiduals(dA, dB math.Vector3, angle float64) []float64 {
	cos := dA.Dot(dB) / (dA.Length() * dB.Length())
	return []float64{cos - stdmath.Cos(angle)}
}

// directedResiduals signs the angle about an implied axis fixed by the reference configuration:
// the axis is captured (normalised dA×dB, or a perpendicular of dA when the two start parallel) on
// the first evaluation and held constant thereafter, so the signed angle stays continuous and can
// pass through 180°.
func (c *AngleConstraint) directedResiduals(dA, dB math.Vector3) []float64 {
	if !c.impliedSet {
		c.implied = impliedAngleAxis(dA, dB)
		c.impliedSet = true
	}
	return signedAngleResiduals(dA, dB, c.implied, c.angle)
}

// signedAngleResiduals returns the wrapped difference between the signed angle from dA to dB about
// axis and the target. The signed angle is atan2((dA×dB)·axiŝ, dA·dB) — scale-invariant in dA, dB
// — and the residual is wrapped into (−π, π] so a target past ±180° is reached at its physically
// equivalent configuration. A degenerate axis falls back to the unsigned cosine residual.
func signedAngleResiduals(dA, dB, axis math.Vector3, target float64) []float64 {
	unit, err := math.UnitVector3FromVector(axis)
	if err != nil {
		return undirectedAngleResiduals(dA, dB, target)
	}
	sinComp := dA.Cross(dB).Dot(unit.AsVector())
	cosComp := dA.Dot(dB)
	signed := stdmath.Atan2(sinComp, cosComp)
	return []float64{wrapToPi(signed - target)}
}

// impliedAngleAxis is the reference axis a directed angle signs about: the direction of dA×dB, or —
// when dA and dB start parallel (a near-zero cross product) — a stable perpendicular of dA, so the
// solve still has a well-defined rotation sense.
func impliedAngleAxis(dA, dB math.Vector3) math.Vector3 {
	if unit, err := math.UnitVector3FromVector(dA.Cross(dB)); err == nil {
		return unit.AsVector()
	}
	perp, _ := tangentFrame(dA)
	return perp
}

// wrapToPi wraps an angle difference into (−π, π] so the signed residual is zero at every 2π
// multiple of the target (the mechanism that lets a directed/reference-vector angle pass 180°).
func wrapToPi(x float64) float64 {
	return stdmath.Atan2(stdmath.Sin(x), stdmath.Cos(x))
}
