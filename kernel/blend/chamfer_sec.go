// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/math"
)

// TwoDistanceChamfer is an asymmetric chamfer: setback D1 on support 1 and D2 on support 2 (OCCT
// ChFiDS_TwoDist). Extent reports the larger setback, which sizes the max-setback validity bound.
type TwoDistanceChamfer struct {
	D1, D2 float64
}

// IsChamfer is true: a chamfer section is a straight chord.
func (TwoDistanceChamfer) IsChamfer() bool { return true }

// Extent returns the larger of the two setbacks.
func (c TwoDistanceChamfer) Extent(float64) float64 { return stdmath.Max(c.D1, c.D2) }

// DistanceAngleChamfer is a chamfer set by a setback D1 on support 1 and Angle measured from
// support 1 (OCCT ChFiDS_DistAngle); the second setback is D1·tan(Angle).
type DistanceAngleChamfer struct {
	D1, Angle float64
}

// IsChamfer is true.
func (DistanceAngleChamfer) IsChamfer() bool { return true }

// Extent returns the larger of the setback and its angle-derived counterpart.
func (c DistanceAngleChamfer) Extent(float64) float64 {
	return stdmath.Max(c.D1, c.D1*stdmath.Tan(c.Angle))
}

// D2 returns the second setback D1·tan(Angle), the chamfer's contact distance on support 2.
func (c DistanceAngleChamfer) D2() float64 { return c.D1 * stdmath.Tan(c.Angle) }

// ChamferSection is a chamfer's straight cross-section: the chord from the contact on support 1
// (FootA, v=0) to the contact on support 2 (FootB, v=1). Unlike a fillet's circular arc, the
// chamfer surface is ruled between these two contact curves (mirrors OCCT ChFiDS_ChamfSpine).
type ChamferSection struct {
	FootA, FootB math.Point3
}

// PointAt returns the chord point at v∈[0,1]: FootA at 0, FootB at 1.
func (s ChamferSection) PointAt(v float64) math.Point3 {
	return s.FootA.TranslateBy(s.FootA.VectorTo(s.FootB).Scale(math.Scalar(v)))
}

// chamferSectionAt places the chamfer chord at edge point e: the contact on each support recedes
// its setback distance along that support's in-face inward perpendicular (mA, mB — unit, ⟂ the
// edge, into each face), exactly as the two-plane chamfer cuts back from the edge.
func chamferSectionAt(e math.Point3, mA, mB math.Vector3, d1, d2 float64) ChamferSection {
	return ChamferSection{
		FootA: e.TranslateBy(mA.Scale(math.Scalar(d1))),
		FootB: e.TranslateBy(mB.Scale(math.Scalar(d2))),
	}
}
