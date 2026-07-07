// SPDX-License-Identifier: GPL-2.0-only

package blend

// EvolRadiusFillet is a variable-radius fillet section: a circular arc whose radius follows a
// RadiusLaw along the guide (OCCT ChFiDS_FilSpine carrying an evolutive law). The section shape is
// the same circular arc as the constant-radius fillet (sectionAt) — only the radius the marcher
// samples per station changes, via Extent. The per-station centre solve the varying radius needs
// (the offset distance is no longer constant, so the centre curve is not a single offset∩offset)
// is the marcher's variable-radius generalization; this type carries the law it reads.
type EvolRadiusFillet struct {
	Law RadiusLaw
}

// IsChamfer is false: an evolving fillet is still a circular-arc section.
func (EvolRadiusFillet) IsChamfer() bool { return false }

// Extent returns the law's radius at spine abscissa w.
func (f EvolRadiusFillet) Extent(w float64) float64 { return f.Law.At(w) }
