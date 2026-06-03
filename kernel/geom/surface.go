// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// Surface is a parametrically evaluable 3D surface (the analytic surfaces and
// the NURBS surface satisfy it). A parameter pair (u, v) in the U/V domains maps
// to a point, the two partial-derivative tangents, and a unit normal.
type Surface interface {
	// PointAt returns the position at parameters (u, v).
	PointAt(u, v float64) math.Point3
	// DerivativesAt returns the partials ∂P/∂u and ∂P/∂v at (u, v).
	DerivativesAt(u, v float64) (du, dv math.Vector3)
	// NormalAt returns the unit surface normal (du×dv normalized), or the zero
	// vector at a degenerate point such as a sphere pole or cone apex.
	NormalAt(u, v float64) math.Vector3
	// UDomain and VDomain return the valid parameter ranges; periodic
	// directions (around an axis) return [0, 2π], unbounded ones return ±Inf.
	UDomain() (lo, hi float64)
	VDomain() (lo, hi float64)
	// ParamAt inverts PointAt: for a point on the surface it reproduces PointAt's
	// parameters (angular ones wrapped to [0, 2π)) — exactly for the analytic
	// surfaces, numerically for NURBS. This on-surface inverse is what trimmed-face
	// tessellation needs (loop vertices lie on the face's surface). Off-surface, it
	// returns the frame projection along the parameter directions, which equals the
	// metric nearest point for the plane, cylinder and sphere but not exactly for the
	// cone or torus (where it is still a stable, continuous projection).
	ParamAt(p math.Point3) (u, v float64)
}

// normalFromPartials returns the unit normal du×dv, or zero when the partials
// are parallel/degenerate. Analytic surfaces override this with exact formulas
// where cheaper, but it is the definitional fallback and the test oracle.
func normalFromPartials(du, dv math.Vector3) math.Vector3 {
	return unitOrZero(du.Cross(dv))
}

// fullCircleDomain is the [0, 2π] parameter range shared by every direction
// that wraps around an axis (cylinder/cone angle, sphere longitude, torus).
func fullCircleDomain() (lo, hi float64) { return 0, twoPi }

// unboundedDomain is the parameter range for a direction with no natural limit
// (an infinite plane, the height of an unbounded cylinder/cone).
func unboundedDomain() (lo, hi float64) { return stdmath.Inf(-1), stdmath.Inf(1) }
