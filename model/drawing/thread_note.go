// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"
	"strings"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Thread-aware hole notes (#1995). A hole note reads as a plain diameter (Ø<d>) unless the hole is
// threaded — a machined (cut) thread swaps the bore's cylindrical face for a geom.ThreadedCylinder,
// which carries the authored designation ("M6x1"), handedness and depth. So the note distinguishes a
// tapped hole (rendered as its thread designation) from a plain drilled hole (rendered as Ø<d>),
// re-resolving with the model like every other hole note. The designation is read off the surface,
// not reconstructed from radius+pitch, because metric-vs-imperial and the model-diameter basis are
// ambiguous once only geometry remains (see geom.ThreadedCylinder.Designation).

// threadCallout is one machined-thread face resolved for a hole note: where its axis pierces the
// view plane (model-2D cm — the space recovered hole circles live in) and the callout designation.
type threadCallout struct {
	center      math.Point2
	designation string
}

// threadCalloutsFrom collects a callout for every threaded face in the body, projecting each face's
// axis point onto the view plane so a recovered hole circle can be matched to its thread by centre
// proximity. When the view looks down a hole's axis (the only way its rim projects as a full circle,
// which is what circlesFromProjection recovers), the axis is parallel to the view direction, so the
// projected axis point coincides with the recovered circle's centre.
func threadCalloutsFrom(body *topo.Body, basis hlr.View) []threadCallout {
	var out []threadCallout
	for _, f := range body.Faces() {
		tc, ok := f.Geometry().(geom.ThreadedCylinder)
		if !ok {
			continue
		}
		des := threadDesignationText(tc)
		if des == "" {
			continue
		}
		out = append(out, threadCallout{center: hlr.ProjectPoint(basis, tc.Origin), designation: des})
	}
	return out
}

// threadDesignationText renders a threaded face's callout: its authored designation, with a "- LH"
// suffix appended for a left-handed thread whose designation does not already carry it (mirrors the
// feature's own handedness merge, #1892).
func threadDesignationText(tc geom.ThreadedCylinder) string {
	des := strings.TrimSpace(tc.Designation)
	if des == "" {
		return ""
	}
	if !tc.RightHanded && !strings.Contains(strings.ToUpper(des), "LH") {
		des += " - LH"
	}
	return des
}

// threadAt returns the designation of the threaded face coaxial with a hole recovered at centre with
// the given radius (model-2D cm), or "" when the hole is plain. A thread matches when its projected
// axis point sits within a quarter of the hole radius of the centre — tight enough that distinct
// holes never cross-match, loose enough to absorb circle-fit and projection noise.
func threadAt(threads []threadCallout, center math.Point2, radius float64) string {
	tol := 0.25 * radius
	if tol < 1e-3 {
		tol = 1e-3
	}
	for _, t := range threads {
		if stdmath.Hypot(float64(t.center.X-center.X), float64(t.center.Y-center.Y)) <= tol {
			return t.designation
		}
	}
	return ""
}
