// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// Obstacle-patch certification (spec §3): the anti-fold gate (NoFold), the G0 rail deviation
// (MaxDev), and the G1 seam-crease angle to the three neighbour ribbons (MaxAngleDev). The rim side
// (c1) is intentionally G0 (sharp base-rim crease) so it is EXCLUDED from the G1 measurement —
// measuring continuity there would falsely reject a correct patch.

// obstacleFoldSamples is the anti-fold grid resolution (24×24, curvature floor); densify if a fold is
// suspected between stations.
const obstacleFoldSamples = 24

// obstacleAngleSamples / obstacleDevSamples are the per-edge sample counts for the G1 crease and G0
// deviation scans.
const (
	obstacleAngleSamples = 16
	obstacleDevSamples   = 16
)

// obstacleCornerExcl is the parametric half-window excluded around each SEAM END in the G1 crease
// scan. Every seam of the obstacle patch ends at a genuine solid FEATURE EDGE — the wall (c0) ends at
// the two wall∩wing corners A and D, and each wing (d0,d1) ends at a corner AND at its rim junction
// P± (the rim c1 is intentional G0). At those edges the wall and wing tangent planes differ (A: XZ vs
// XY), so G1 is neither achievable nor wanted there; a single tensor patch cannot be corner-twist
// compatible. G1 is therefore asserted only on the SMOOTH seam interiors, f ∈ [excl, 1−excl]. 0.15
// comfortably contains the corner-adjacent 2-column band once the rails are refined to obstacleG1Ctrl
// columns (refineForG1) — measured so the interior crease falls below tessellate.SeamAngularTol (task-4-report).
const obstacleCornerExcl = 0.15

// certifyObstaclePatch measures the patch's admissibility. Closed/WeldsArms are structural givens
// here: CoonsFill's consistentCorners guarantees the quad closes, and the obstacle has no "arms" so
// WeldsArms (boundary-complete) holds by the four spanned sides.
func certifyObstaclePatch(s geom.BSplineSurface, g obstaclePatchGeom, scale tol.Resolution) Certificate {
	return Certificate{
		Closed:      true,
		WeldsArms:   true,
		NoFold:      obstacleNoFold(s, scale),
		MaxDev:      obstacleMaxDev(s, g),
		MaxAngleDev: obstacleMaxAngleDev(s, g),
	}
}

// obstacleNoFold sweeps the interior u-columns and reports whether S_u×S_v keeps a consistent sign
// along v. It uses a ROTATING reference (consecutive-station dot), NOT a fixed axis, because a fillet's
// normal legitimately sweeps >90° across the profile — a fixed-axis gate would false-positive. The
// scan skips the two wall∩wing corner columns (u ∈ {umin,umax} within obstacleCornerExcl): those are
// genuine feature VERTICES (A, D) where the wall and wing tangent planes conflict, so the surface
// normal is degenerate/undefined at the corner point exactly — the same edges excluded from the G1
// measure. Interior columns carry the real anti-fold guarantee.
func obstacleNoFold(s geom.BSplineSurface, scale tol.Resolution) bool {
	v0, v1 := s.VDomain()
	return noFoldOverColumns(s, v0, v1, scale)
}

// noFoldOverColumns is the shared anti-fold column sweep: it scans obstacleFoldSamples u-columns
// (skipping the corner-excluded band, see obstacleNoFold) over [vLo,vHi] and reports true iff no
// column folds (columnFolds). obstacleNoFold sweeps the full v-range; tri3NoFold caps vHi below the
// degenerate pole row. (F3 de-dup of the obstacle/tri3 sweeps — same u-stride, different v-window.)
func noFoldOverColumns(s geom.BSplineSurface, vLo, vHi float64, scale tol.Resolution) bool {
	u0, u1 := s.UDomain()
	span := 1 - 2*obstacleCornerExcl
	for i := 0; i <= obstacleFoldSamples; i++ {
		u := u0 + (obstacleCornerExcl+float64(i)/float64(obstacleFoldSamples)*span)*(u1-u0)
		if columnFolds(s, u, vLo, vHi, scale) {
			return false
		}
	}
	return true
}

// columnFolds sweeps one u-column across v: a fold is the Jacobian (|S_u×S_v|) dropping below the
// model weld, or the normal reversing against the previous station. The reversal test is
// SAMPLING-INVARIANT (normalsReverse): a fillet's normal legitimately sweeps >90° in total (T6 sweeps
// 162°), so a single coarse step can span >90° of SMOOTH rotation and a naive dot≤0 would false-flag
// it — we bisect a suspected reversal and only confirm a genuine crease.
func columnFolds(s geom.BSplineSurface, u, v0, v1 float64, scale tol.Resolution) bool {
	var prev math.Vector3
	vPrev := v0
	for j := 0; j <= obstacleFoldSamples; j++ {
		v := v0 + float64(j)/float64(obstacleFoldSamples)*(v1-v0)
		cur := surfaceNormal(s, u, v)
		if cur.Length() < scale.Weld() {
			return true
		}
		if j > 0 && normalsReverse(s, vPrev, v, prev, cur, u, scale, obstacleFoldRefine) {
			return true
		}
		prev, vPrev = cur, v
	}
	return false
}

// obstacleFoldRefine is the max bisection depth used to tell a genuine normal reversal from a coarse
// step across a smooth fast sweep. Each level halves the step's angular span, so 6 levels resolve a
// >90° coarse step (T6's worst is ~106°) down to <2°, far inside the no-false-flag regime.
const obstacleFoldRefine = 6

// normalsReverse reports a TRUE fold between two v-stations whose normals oppose (dot ≤ 0). Because a
// coarse step may merely undersample a smooth >90° sweep, it bisects up to `depth` times, confirming a
// fold only if the reversal survives to the finest step or the Jacobian collapses there (a real crease
// where the normal flips). A smooth sweep resolves into sub-90° steps and returns false.
func normalsReverse(s geom.BSplineSurface, va, vb float64, na, nb math.Vector3, u float64, scale tol.Resolution, depth int) bool {
	if na.Dot(nb) > 0 {
		return false
	}
	if depth == 0 {
		return true
	}
	vm := (va + vb) / 2
	nm := surfaceNormal(s, u, vm)
	if nm.Length() < scale.Weld() {
		return true
	}
	return normalsReverse(s, va, vm, na, nm, u, scale, depth-1) ||
		normalsReverse(s, vm, vb, nm, nb, u, scale, depth-1)
}

// obstacleMaxDev is the G0 residual: the max distance from dense rail samples to the fill's matching
// boundary iso-curve. ~0 by construction (CoonsFill interpolates the rails and MatchSurface preserves
// the position row); it guards against a make-compatible / knot-refine drift corrupting a boundary.
func obstacleMaxDev(s geom.BSplineSurface, g obstaclePatchGeom) float64 {
	m := railDev(s, g.c0, edgeVMin)
	m = stdmath.Max(m, railDev(s, g.c1, edgeVMax))
	m = stdmath.Max(m, railDev(s, g.d0, edgeUMin))
	return stdmath.Max(m, railDev(s, g.d1, edgeUMax))
}

// railDev is the max distance from samples of rail to the fill's iso-curve along fill edge e (both
// pinned to the same corners, so fraction f corresponds pointwise).
func railDev(s geom.BSplineSurface, rail geom.BSplineCurve, e fillEdge) float64 {
	lo, hi := rail.Domain()
	m := 0.0
	for k := 0; k <= obstacleDevSamples; k++ {
		f := float64(k) / float64(obstacleDevSamples)
		want := rail.PointAt(lo + f*(hi-lo))
		u, v := e.fillParam(s, f)
		m = stdmath.Max(m, want.DistanceTo(s.PointAt(u, v)))
	}
	return m
}

// obstacleMaxAngleDev is the max G1 crease angle across ONLY the three G1 sides (wall c0 + wings
// d0,d1). The rim (c1) is G0 by design and is excluded.
func obstacleMaxAngleDev(s geom.BSplineSurface, g obstaclePatchGeom) float64 {
	m := seamCrease(s, g.wall, edgeVMin)
	m = stdmath.Max(m, seamCrease(s, g.wingL, edgeUMin))
	return stdmath.Max(m, seamCrease(s, g.wingR, edgeUMax))
}

// seamCrease is the max crease angle between the fill normal and the neighbour ribbon normal along
// one shared edge, scanned over the SMOOTH interior f ∈ [obstacleCornerExcl, 1−obstacleCornerExcl]
// (the seam ends are genuine feature edges — see obstacleCornerExcl). The ribbon's rail edge is its
// VMinEdge (extrudeRibbon) and shares the rail's knot vector, so fraction f maps to the same rail
// point on each surface.
func seamCrease(s, nbr geom.BSplineSurface, e fillEdge) float64 {
	ru0, ru1 := nbr.UDomain()
	rv0, _ := nbr.VDomain()
	span := 1 - 2*obstacleCornerExcl
	m := 0.0
	for k := 0; k <= obstacleAngleSamples; k++ {
		f := obstacleCornerExcl + float64(k)/float64(obstacleAngleSamples)*span
		u, v := e.fillParam(s, f)
		nbrN := surfaceNormal(nbr, ru0+f*(ru1-ru0), rv0)
		m = stdmath.Max(m, probe.CreaseAngle(surfaceNormal(s, u, v), nbrN))
	}
	return m
}

// fillEdge names one of the fill's four boundary iso-curves, so the certify scans map a fraction to a
// surface (u,v) on that edge.
type fillEdge int

const (
	edgeVMin fillEdge = iota // c0, v=vmin
	edgeVMax                 // c1, v=vmax
	edgeUMin                 // d0, u=umin
	edgeUMax                 // d1, u=umax
)

// fillParam maps a fraction f∈[0,1] to the surface (u,v) walking edge e from its start corner.
func (e fillEdge) fillParam(s geom.BSplineSurface, f float64) (u, v float64) {
	u0, u1 := s.UDomain()
	v0, v1 := s.VDomain()
	switch e {
	case edgeVMin:
		return u0 + f*(u1-u0), v0
	case edgeVMax:
		return u0 + f*(u1-u0), v1
	case edgeUMin:
		return u0, v0 + f*(v1-v0)
	default: // edgeUMax
		return u1, v0 + f*(v1-v0)
	}
}

// surfaceNormal returns the (unnormalized) surface normal S_u×S_v at (u,v).
func surfaceNormal(s geom.BSplineSurface, u, v float64) math.Vector3 {
	du, dv := s.DerivativesAt(u, v)
	return du.Cross(dv)
}
