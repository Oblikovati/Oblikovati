// SPDX-License-Identifier: GPL-2.0-only

package blend

// SectionFunctional is the pluggable cross-section that makes a fillet and a chamfer the SAME
// pipeline differing only by their section (ADR-0050) — OCCT's Blend_Function / BlendFunc_ConstRad
// seam. The marcher (Phase 4) sweeps this section along the spine, Newton-solving the foot points
// so the section stays tangent to both support surfaces; the section family (circular arc for a
// fillet, straight chord for a chamfer) decides the residual it solves.
//
// Phase 3 lands the port and its parameter carriers; the residual/Jacobian methods the marcher
// needs are added by Phase 4 (a SectionSolver sub-interface), so nothing here computes geometry yet.
type SectionFunctional interface {
	// IsChamfer reports the section family: true ⇒ a straight chord (chamfer), false ⇒ a circular
	// arc tangent to both walls (fillet).
	IsChamfer() bool
	// Extent returns the section's governing size at spine abscissa w — the fillet radius, or the
	// chamfer's setback on side 1. It sizes the seed section and feeds the max-radius validity bound.
	Extent(w float64) float64
}

// ConstRadiusFillet is the constant-radius rolling-ball fillet section: a circular arc of radius R
// tangent to both support surfaces. The variable-radius law is Phase 5 (EvolRadiusSection).
type ConstRadiusFillet struct {
	R float64
}

// IsChamfer is false: a fillet section is a circular arc.
func (ConstRadiusFillet) IsChamfer() bool { return false }

// Extent returns the constant radius.
func (f ConstRadiusFillet) Extent(float64) float64 { return f.R }

// SymmetricChamfer is the equal-setback chamfer section: a straight chord cutting distance D back
// along each support face. The two-distance and distance-angle modes are Phase 5 (ChamferSection).
type SymmetricChamfer struct {
	D float64
}

// IsChamfer is true: a chamfer section is a straight chord.
func (SymmetricChamfer) IsChamfer() bool { return true }

// Extent returns the setback distance.
func (c SymmetricChamfer) Extent(float64) float64 { return c.D }
