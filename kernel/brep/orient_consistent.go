// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// orientFaceSigns derives a CONSISTENT outward orientation sign (±1) for each prepared face, INDEPENDENT
// of the stored Face.Reversed flag. The winding number needs every face signed so its normal points out
// of the body; an imported B-rep's Reversed flags can be inconsistent (a STEP extrusion→quadric normal-
// side defect), which makes the signed solid angles cancel and a solid interior read as outside.
//
// The reliable orientation carrier on a valid B-rep is the LOOP traversal (STEP ORIENTED_EDGE), not the
// face normal-side flag: on a consistently-oriented shell every shared edge is walked in opposite
// directions by its two faces, so each face's outer-loop handedness about its geometric normal (S_u×S_v)
// already defines one global orientation. The sign of the outer loop's signed area in (u, v) is that
// handedness (+1 CCW about S_u×S_v → normal is outward, −1 → inward). One global flip, chosen so the
// enclosed signed volume is positive, orients the whole shell outward.
func orientFaceSigns(faces []fluxFace) []float64 {
	signs := make([]float64, len(faces))
	volume := 0.0
	for i := range faces {
		signs[i] = loopHandedness(faces[i].region)
		volume += signs[i] * faceVolumeTerm(&faces[i])
	}
	if volume < 0 { // the loop handedness oriented the shell inward — flip the whole body outward
		for i := range signs {
			signs[i] = -signs[i]
		}
	}
	return signs
}

// loopHandedness is the sign of the outer ring's signed area in (u, v): +1 when the ring runs CCW about
// the surface normal S_u×S_v (so that normal is the outward one), −1 when CW — NEGATED when the face is
// the ring's COMPLEMENT, because a ring that runs CCW around the region it encloses runs CW around
// everything else on the domain. Without that negation the big spherical cap of a ball joined with a rod
// took the inward normal as outward, and the ball's own centre read outside its solid
// (Oblikovati/Oblikovati#3453, #3429). A boundaryless face (a whole sphere/torus, its own closed body)
// has no ring to read; it returns +1 and leans on the volume sign.
func loopHandedness(r trimRegion) float64 {
	if len(r.rings) == 0 {
		return 1
	}
	sign := 1.0
	if regionSignedArea(r) < 0 {
		sign = -1
	}
	if r.complement {
		return -sign
	}
	return sign
}

// regionSignedArea is the signed area the face's rings enclose in (u, v).
//
// A ring that closes in the parameter plane contributes its own shoelace, and they are SUMMED: a hole
// is wound against its enclosing ring, so it subtracts, and the total keeps the outer ring's sign
// without having to decide which ring is outer.
//
// A ring that does NOT close there is a band rim — a closed circuit in 3-D, but an open polyline in the
// covering space. Its shoelace says nothing: a rim at constant v shoelaces to ZERO whichever way the
// band is oriented, which is what made a plate's bore wall report the handedness of its own opposite
// and integrate the bore as added rather than subtracted, 564.36 against a true 354.92. Those rims are
// joined into ONE circuit before the area is taken. A band is bounded by its two rims and the two seam
// segments, and concatenating the rims lets the shoelace supply both seams itself: the junction between
// the rims is one, the closing chord of the concatenation is the other.
//
// This is what OCCT gets for free. It stores a seam edge EXPLICITLY — twice in the wire, with two
// pcurves — so a band's wire is already a closed contour and ShapeAnalysis::TotCross2D applies to it
// unchanged. Our bands carry no seam edge, so the circuit is reassembled here
// (Oblikovati/Oblikovati#3506).
func regionSignedArea(r trimRegion) float64 {
	area := 0.0
	var wrapping []math.Point2
	for _, ring := range r.rings {
		if !ringSpansAPeriod(ring, r) {
			area += signedArea2D(ring)
			continue
		}
		wrapping = append(wrapping, ring...)
	}
	if len(wrapping) < 3 {
		return area
	}
	return area + signedArea2D(wrapping)
}

// ringSpansAPeriod reports whether a ring travels a WHOLE TURN in a periodic parameter instead of
// returning to where it started — what makes it an open polyline in the covering space. A face on a
// non-periodic surface can never do it, and neither can a ring that walks out along a seam and back
// (a full cone's side loop), whose net travel is zero.
func ringSpansAPeriod(ring []math.Point2, r trimRegion) bool {
	if len(ring) < 3 {
		return false
	}
	if r.uPeriodic && ringClosesByAWholeTurn(ring, func(p math.Point2) float64 { return float64(p.X) }) {
		return true
	}
	return r.vPeriodic && ringClosesByAWholeTurn(ring, func(p math.Point2) float64 { return float64(p.Y) })
}

// ringClosesByAWholeTurn reports whether the ring's CLOSING chord — from its last sample back to its
// first, the one edge the sample list leaves implicit — jumps a whole turn rather than one sampling
// step.
//
// It is judged against the ring's OWN widest interior step, so it assumes nothing about how densely
// the ring was sampled. A rim's closing chord is the whole turn less one step, which is many times the
// widest step; a ring that genuinely closes has a closing chord the size of its neighbours. Comparing
// against a fixed fraction of the period instead would misread a sparsely sampled patch that happens
// to reach far in u.
func ringClosesByAWholeTurn(ring []math.Point2, at func(math.Point2) float64) bool {
	closing, widest := stdmath.Abs(at(ring[len(ring)-1])-at(ring[0])), 0.0
	for i := 1; i < len(ring); i++ {
		widest = stdmath.Max(widest, stdmath.Abs(at(ring[i])-at(ring[i-1])))
	}
	return closing > twoPi/2 && closing > 2*widest
}

// twoPi is one full turn, the period of every angular surface parameter in the kernel.
const twoPi = 2 * stdmath.Pi

// volumeGridSteps is the fixed (u, v) sampling per face for the signed-volume sign probe. The probe needs
// only the SIGN of the body volume to fix the global orientation, so a coarse uniform grid — no adaptive
// refinement, no singularity to resolve — is enough.
const volumeGridSteps = 8

// faceVolumeTerm is the face's contribution to 3·V, the divergence-theorem volume ∫∫ P·(S_u×S_v) du dv
// over its trimmed domain (with the loop-handedness sign applied by the caller). A coarse uniform grid
// weighted by the in-trim fraction; only its sign, summed over the body, is used.
func faceVolumeTerm(f *fluxFace) float64 {
	du := (f.u1 - f.u0) / volumeGridSteps
	dv := (f.v1 - f.v0) / volumeGridSteps
	sum := 0.0
	for i := 0; i < volumeGridSteps; i++ {
		for j := 0; j < volumeGridSteps; j++ {
			u0, v0 := f.u0+float64(i)*du, f.v0+float64(j)*dv
			frac := cellTrimFraction(f.region, u0, u0+du, v0, v0+dv)
			if frac == 0 {
				continue
			}
			sum += volumeIntegrand(f.cf.surface, u0+0.5*du, v0+0.5*dv) * du * dv * frac
		}
	}
	return sum
}

// volumeIntegrand evaluates P·(S_u×S_v) at (u, v) — the outward flux of the position field, whose surface
// integral is three times the enclosed volume (Gauss divergence of F = P).
func volumeIntegrand(s geom.Surface, u, v float64) float64 {
	du, dv := s.DerivativesAt(u, v)
	n := du.Cross(dv)
	p := s.PointAt(u, v)
	return float64(p.AsVector().Dot(n))
}
