// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The ONE surface–surface intersection the tolerant analytic boolean uses (ADR-0058). OCCT's
// IntTools_FaceFace dispatches a face pair to a closed-form intersector when the surface pair admits one
// and to a general walking-line marcher otherwise, bounding the search to the faces' parameter ranges.
// SurfaceIntersect is that entry over geom.Surface: it prefers the EXACT closed form and falls to the
// general predictor–corrector tracer, clipped to the operands' region of interest, for every pair the
// closed form does not solve.

// SurfaceIntersect returns the intersection curves of two surfaces as Curve3s. It first tries the exact
// closed form (IntersectSurfacesAnalytic — plane∩plane → a line, plane∩quadric → a conic, each
// conditioning-gated inside: an oblique or ill-conditioned pair reports handled=false there). Where no
// closed form applies it marches the joint zero over box with the general tracer and returns the traced
// curve. handled is false only when neither yields a curve and the surfaces may still cross — the caller
// declines to the mesh rescue. box bounds the search so an UNBOUNDED surface (plane/cylinder/cone) is
// intersected only where the two faces overlap, exactly as OCCT clips IntTools_FaceFace to the faces' UV
// ranges; build it from the operands' bounding box, e.g. face1.RangeBox().Union(face2.RangeBox()).
//
// Example — two equal perpendicular cylinders (the Steinmetz bicylinder) yield their two saddle curves:
//
//	cz := geom.Cylinder{Origin: math.P3(0, 0, 0), AxisDir: math.V3(0, 0, 1).AsUnit(), Radius: 1}
//	cx := geom.Cylinder{Origin: math.P3(0, 0, 0), AxisDir: math.V3(1, 0, 0).AsUnit(), Radius: 1}
//	curves, _ := geom.SurfaceIntersect(cz, cx, math.NewBox(math.P3(-2, -2, -2), math.P3(2, 2, 2)), res)
func SurfaceIntersect(a, b Surface, box math.Box, res Resolution) (curves []Curve3, handled bool) {
	if cs, ok := IntersectSurfacesAnalytic(a, b, res); ok {
		return cs, true // exact closed form (an empty result = a known non-crossing / tangent touch)
	}
	// No closed form: march. Try each operand as the base — the tracer walks the base's parameter
	// window, and a base whose domain the box bounds more tightly seeds the continuation better.
	for _, roles := range [2][2]Surface{{a, b}, {b, a}} {
		if cs := marchedCurves(roles[0], roles[1], box); len(cs) > 0 {
			return cs, true
		}
	}
	return nil, false
}

// marchedCurves traces base∩other over the box-derived parameter window and wraps each traced polyline
// as a Curve3 (a Polyline — a valid, if non-analytic, boundary curve; recognising it as an analytic
// circle/ellipse is a later refinement, ADR-0058 phase 1b).
func marchedCurves(base, other Surface, box math.Box) []Curve3 {
	traced := TraceSurfaceIntersection(base, other, SurfaceWindow(base, box))
	if traced.Declined {
		// The continuation spent its corrector budget: what it has is a PARTIAL trace, and shipping a
		// partial curve set as if it were the whole intersection is the silent degradation the ground
		// rules forbid. Report nothing, so SurfaceIntersect answers handled=false and the caller takes
		// its own named decline (#3477).
		return nil
	}
	out := make([]Curve3, 0, len(traced.Curves))
	for _, poly := range traced.Curves {
		// Stamp the trace's achieved deviation on every curve it produced: a marched Curve3 is a chord
		// approximation, and the edge built from it must be able to say how exact it is (#3489).
		if c, err := NewMarchedPolyline(poly, traced.Deviation); err == nil {
			out = append(out, c)
		}
	}
	return out
}

// SurfaceWindow derives a surface's marching window from a bounding box: a bounded or periodic
// parameter direction uses its whole domain; an UNBOUNDED direction (a plane/cylinder/cone's axial run)
// is clipped to where the box projects onto it, so the marcher never sweeps the infinite surface. It is
// the general windowing SurfaceIntersect uses internally AND the curved-boolean imprint uses to clip a
// surface to an operand body's own extent (pass that body's RangeBox), replacing the per-primitive
// apex/axial-band windows (ADR-0058 phase 3).
//
//	win := geom.SurfaceWindow(coneSurface, coneBody.RangeBox()) // full angle, apex-distance band of the body
//
// It pads an unbounded direction outward by windowPadFraction so a crossing exactly on a box face is not
// clipped — needed by the general marcher. A caller that wants the box's EXACT projection (a boolean imprint
// windowed to an operand body's own caps, whose downstream assembly is sampling-sensitive — the near-pinch
// band, #1818) uses SurfaceWindowTight instead.
func SurfaceWindow(s Surface, box math.Box) SurfaceGrid {
	return surfaceWindowPadded(s, box, windowPadFraction)
}

// SurfaceWindowTight is SurfaceWindow with NO outward pad: an unbounded direction is clipped to exactly the
// box's projection, so windowing a primitive side to its own body box reproduces that primitive's intrinsic
// axial/apex band bit-for-bit (the exact window the per-primitive boolean imprints used). Sampling-sensitive
// consumers (the near-pinch cut/join assembly, #1818) require this exactness; the padded SurfaceWindow's 5%
// over-sweep shifts the marched loop enough to break their raw-loop weld.
func SurfaceWindowTight(s Surface, box math.Box) SurfaceGrid {
	return surfaceWindowPadded(s, box, 0)
}

// windowPadFraction widens an unbounded marching window past the box projection so a tangency exactly on a
// box face is not clipped (SurfaceWindow / the general marcher). SurfaceWindowTight passes 0.
const windowPadFraction = 0.05

func surfaceWindowPadded(s Surface, box math.Box, padFrac float64) SurfaceGrid {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	uMin, uMax := paramWindow(s, box, uLo, uHi, true, padFrac)
	vMin, vMax := paramWindow(s, box, vLo, vHi, false, padFrac)
	return SurfaceGrid{UMin: uMin, UMax: uMax, VMin: vMin, VMax: vMax}
}

// paramWindow returns the [min,max] window for one parameter direction: a finite domain as-is (a bounded
// or periodic direction), or — for an unbounded direction — the box's projection onto it (via ParamAt of
// the corners), padded so a crossing on the box boundary is not clipped.
func paramWindow(s Surface, box math.Box, lo, hi float64, isU bool, padFrac float64) (float64, float64) {
	if !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
		return lo, hi
	}
	pmin, pmax := stdmath.Inf(1), stdmath.Inf(-1)
	for _, c := range box.Corners() {
		u, v := s.ParamAt(c)
		p := u
		if !isU {
			p = v
		}
		pmin, pmax = stdmath.Min(pmin, p), stdmath.Max(pmax, p)
	}
	pad := padFrac * (pmax - pmin) // widen so a tangency exactly on a box face stays inside the window
	return pmin - pad, pmax + pad
}
