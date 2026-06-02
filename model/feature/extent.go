// SPDX-License-Identifier: GPL-2.0-only

package feature

// ExtentType is how a sketched feature terminates — the PartFeatureExtentEnum.
type ExtentType uint8

const (
	// DistanceExtent: a fixed (possibly parametric) distance.
	DistanceExtent ExtentType = iota
	// ToNextExtent: up to the next face encountered.
	ToNextExtent
	// ThroughAllExtent: all the way through the existing material.
	ThroughAllExtent
	// ToFaceExtent: up to a selected face/work-plane.
	ToFaceExtent
	// FromToExtent: between two selected faces/work-planes.
	FromToExtent
)

// ExtentDirection is which way a feature grows — the PartFeatureExtentDirectionEnum.
type ExtentDirection uint8

const (
	// PositiveDir grows along the profile-plane normal.
	PositiveDir ExtentDirection = iota
	// NegativeDir grows opposite the normal.
	NegativeDir
	// SymmetricDir grows half each way.
	SymmetricDir
)

// Extent describes a sketched feature's termination. Distance is a closure so the
// extent tracks its driving parameter; the to-face/from-to reference inputs are
// modeled as [Ref]s, resolved at recompute (used once those terminations generate
// geometry, kernel phase B+).
type Extent struct {
	Type      ExtentType
	Direction ExtentDirection
	Distance  func() float64
	ToFace    Ref
	FromFace  Ref
}

// signedDistance returns the growth distance for the positive side, honoring the
// direction (symmetric returns the half-distance applied each way by the caller).
func (e Extent) distance() float64 {
	if e.Distance == nil {
		return 0
	}
	return e.Distance()
}
