// SPDX-License-Identifier: GPL-2.0-only

package blend

// ErrorStatus localizes a blend failure to the stage that produced it, mirroring OCCT
// ChFiDS_ErrorStatus. Carrying the stage lets a single bad segment or corner yield a partial
// (Sick-with-hole) result that names what failed, instead of a global panic (ADR-0050).
type ErrorStatus uint8

const (
	// StatusOk is a fully built blend.
	StatusOk ErrorStatus = iota
	// StatusStartSectionFailed: no valid seed cross-section exists — the radius exceeds what the
	// local geometry admits (the planar closed form of this is Phase 2's max-radius check).
	// OCCT ChFiDS_StartsolFailure.
	StatusStartSectionFailed
	// StatusWalkingFailed: the marcher lost the section mid-guide (a foot point left every support
	// or the corrector diverged). OCCT ChFiDS_WalkingFailure.
	StatusWalkingFailed
	// StatusTwistedSurface: the approximated blend surface self-folded. OCCT ChFiDS_TwistedSurface.
	StatusTwistedSurface
	// StatusNotImplemented: a case the engine does not yet build (the Phase-3 marcher stub).
	StatusNotImplemented
)

// String returns a stable diagnostic name.
func (s ErrorStatus) String() string {
	switch s {
	case StatusOk:
		return "ok"
	case StatusStartSectionFailed:
		return "start-section-failed"
	case StatusWalkingFailed:
		return "walking-failed"
	case StatusTwistedSurface:
		return "twisted-surface"
	case StatusNotImplemented:
		return "not-implemented"
	default:
		return "unknown"
	}
}

// Result is the blend engine's output: the segments it built and the localized status. It follows
// OCCT's partial-result contract — HasResult reports a usable (possibly incomplete) build, BadShape
// a build that produced nothing usable — so the feature can go Sick-with-hole rather than fail hard.
type Result struct {
	Segments []BlendSegment
	Status   ErrorStatus
}

// HasResult reports that at least one segment was built (a usable, possibly partial, blend).
func (r Result) HasResult() bool { return len(r.Segments) > 0 }

// BadShape reports a failed build that produced no usable geometry.
func (r Result) BadShape() bool { return r.Status != StatusOk && len(r.Segments) == 0 }
