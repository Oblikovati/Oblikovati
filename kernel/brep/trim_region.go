// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A face's region in its surface's (u, v) domain, as the flux quadrature and the orientation probe both
// have to read it: the unwrapped trim rings PLUS which side of them the face is.
//
// A ring alone does not name a region on a CLOSED parameter domain (a sphere, a torus). Both sides of it
// are bounded there, so a face is as often the ring's COMPLEMENT as its interior — a ball joined with a
// rod through its top keeps the 9/10-of-the-sphere cap BELOW the seam, and that cap's only ring is the
// small circle around the rod. Reading the ring as the region inverted BOTH readings the winding path
// takes off it: the outward sign (a ring that runs CCW around the small cap runs CW around the big one)
// and the quadrature rectangle (the ring's bounding box covers the small cap). So the ball's own centre
// classified as OUTSIDE its own solid, and the boolean that built it was demoted to a faceted fallback
// even though its faces were exact (Oblikovati/Oblikovati#3453, #3429).
//
// On an OPEN domain the question cannot arise: the region outside the rings runs to the domain's own
// boundary or to infinity, so a trimmed face is always the rings' interior, and [pointInTrimUV] reads it
// that way by construction. The side is therefore decided ONCE per face at preparation — never per
// query — by asking the authoritative trim classifier about a single probe deep inside the rings, and
// only on the closed domains where that classifier has an independent answer to give.
type trimRegion struct {
	rings      [][]math.Point2
	complement bool
	// uPeriodic/vPeriodic say whether the surface's own parameter wraps, which is what makes a ring
	// able to be an OPEN polyline in the covering space rather than a closed contour (see
	// regionSignedArea).
	uPeriodic, vPeriodic bool
}

// contains is the region's own point membership in (u, v): the rings' even-odd interior, or everything
// but it. A boundaryless face (a whole sphere/torus) owns its entire domain.
func (r trimRegion) contains(q math.Point2) bool {
	if len(r.rings) == 0 {
		return true
	}
	return pointInLoops2D(r.rings, q) != r.complement
}

// faceTrimRegion projects a face's trim into (u, v) once — loopToUV inverts ParamAt per sample, so this
// must not run per query — and decides which side of those rings the face is.
func faceTrimRegion(f curvedFace) trimRegion {
	uPer, vPer := surfacePeriodic(f.surface)
	return trimRegion{
		rings: trimPolys(f, uPer, vPer), complement: faceIsRingComplement(f, uPer, vPer),
		uPeriodic: uPer, vPeriodic: vPer,
	}
}

// trimPolys projects every loop of the face into one continuous (u, v) polyline (reusing loopToUV's seam
// unwrapping). A boundaryless face returns nil — its whole finite domain is integrated.
func trimPolys(f curvedFace, uPer, vPer bool) [][]math.Point2 {
	if len(f.loops) == 0 {
		return nil
	}
	polys := make([][]math.Point2, 0, len(f.loops))
	for _, loop := range f.loops {
		if poly := loopToUV(f.surface, loop, uPer, vPer); len(poly) >= 3 {
			polys = append(polys, poly)
		}
	}
	return polys
}

// faceIsRingComplement reports whether the face owns everything EXCEPT the region its rings enclose.
//
// It can be true only on a closed parameter domain — one with no axis reaching an exterior ([castAxis])
// — because that is the only domain on which the rings' interior and its complement are both admissible
// regions; everywhere else the region runs to the domain's own boundary and a trimmed face is always the
// rings' interior. Which of the two it is, is a TOPOLOGICAL datum, not a geometric one:
// [curvedFace.outerless] is set exactly when the face's boundary is holes only, so the face wraps the
// whole closed surface minus them (Oblikovati#1406), and topo carries it on the loop through
// topo.LoopSpec/Loop.IsOuter.
//
// It used to be probed instead — the point deepest inside the rings, classified by [pointInTrimUV] —
// which on a closed domain resolved to a reading of the loop's traversal handedness. Handedness orients
// a shell only up to one global sign (orient_consistent.go picks that sign from the body's signed
// volume), so on a body carrying the inverted-but-coherent choice the probe answered "complement" for
// every sphere and torus face (Oblikovati/Oblikovati#3477).
func faceIsRingComplement(f curvedFace, uPer, vPer bool) bool {
	if _, ok := castAxis(f.surface, uPer, vPer); ok {
		return false
	}
	return f.outerless
}

// fluxDomain is the (u, v) rectangle the quadrature covers: the rings' unwrapped bounding box when the
// face is the region they enclose, and the surface's own finite domain when the face is boundaryless or
// is the rings' COMPLEMENT — where that bounding box covers exactly the wrong region. It fails
// (ok=false) only for a face whose domain is unbounded and whose region is not the rings' interior,
// which a closed body never has.
func fluxDomain(f curvedFace, r trimRegion) (u0, u1, v0, v1 float64, ok bool) {
	if len(r.rings) > 0 && !r.complement {
		return polyBounds(r.rings)
	}
	return surfaceDomainRect(f.surface, r)
}

// surfaceDomainRect is the surface's whole finite domain, with each PERIODIC axis re-centred on the
// rings' own branch: loopToUV unwraps a ring onto whichever turn it started on, which the canonical
// [0, 2π] window need not contain, and the quadrature's ring test reads only that branch.
func surfaceDomainRect(s geom.Surface, r trimRegion) (u0, u1, v0, v1 float64, ok bool) {
	centre := ringBranchCentre(r)
	uPer, vPer := surfacePeriodic(s)
	ul, uh := s.UDomain()
	vl, vh := s.VDomain()
	u0, u1 = axisWindow(ul, uh, uPer, centre.X)
	v0, v1 = axisWindow(vl, vh, vPer, centre.Y)
	return u0, u1, v0, v1, isFiniteRect(u0, u1, v0, v1)
}

// axisWindow is one axis of that rectangle: a full turn centred on the rings' branch when the axis is
// periodic, else the surface's own domain.
func axisWindow(lo, hi float64, periodic bool, centre float64) (float64, float64) {
	if !periodic {
		return lo, hi
	}
	return centre - stdmath.Pi, centre + stdmath.Pi
}

// ringBranchCentre is the (u, v) centroid of the first ring — the branch the periodic axes are centred
// on. A boundaryless face has no ring to place, and every full turn is the same window there, so the
// origin serves.
func ringBranchCentre(r trimRegion) math.Point2 {
	if len(r.rings) == 0 {
		return math.P2(0, 0)
	}
	return loopCentroid(r.rings[0])
}

// isFiniteRect reports whether the rectangle is bounded and non-degenerate, the precondition of the
// quadrature.
func isFiniteRect(u0, u1, v0, v1 float64) bool {
	if stdmath.IsInf(u0, 0) || stdmath.IsInf(u1, 0) || stdmath.IsInf(v0, 0) || stdmath.IsInf(v1, 0) {
		return false
	}
	return u1 > u0 && v1 > v0
}
