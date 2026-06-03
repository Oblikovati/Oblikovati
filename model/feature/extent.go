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
	// DistanceFromFaceExtent: a distance measured from a selected face/work-plane.
	DistanceFromFaceExtent
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
// extent tracks its driving parameter. Distance2 (when set) makes the extrude
// asymmetric — Distance grows the positive side, Distance2 the negative side
// (Inventor's two-direction extrude). The to/from terminations reference a work plane
// (resolved like a revolve axis), used by the to-face / from-to / distance-from-face
// modes; the geometry currently requires the target parallel to the sketch plane (a
// flat cap), angled/curved trims being kernel phase C.
type Extent struct {
	Type      ExtentType
	Direction ExtentDirection
	Distance  func() float64
	Distance2 func() float64 // asymmetric second-direction distance (nil ⇒ single direction)
	ToPlane   *WorkPlane     // to-face / distance-from-face target (and the "to" of from-to)
	FromPlane *WorkPlane     // from-to start
}

// distance returns the primary growth distance (0 when unset).
func (e Extent) distance() float64 {
	if e.Distance == nil {
		return 0
	}
	return e.Distance()
}

// distance2 returns the second-direction distance (0 when unset).
func (e Extent) distance2() float64 {
	if e.Distance2 == nil {
		return 0
	}
	return e.Distance2()
}

// isAsymmetric reports whether a distinct second-direction distance is set.
func (e Extent) isAsymmetric() bool { return e.Distance2 != nil }
