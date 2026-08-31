// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
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
		signs[i] = loopHandedness(faces[i].polys)
		volume += signs[i] * faceVolumeTerm(&faces[i])
	}
	if volume < 0 { // the loop handedness oriented the shell inward — flip the whole body outward
		for i := range signs {
			signs[i] = -signs[i]
		}
	}
	return signs
}

// loopHandedness is the sign of the outer loop's signed area in (u, v): +1 when the loop runs CCW about
// the surface normal S_u×S_v (so that normal is the outward one), −1 when CW. A boundaryless face (a whole
// sphere/torus, its own closed body) has no loop to read; it returns +1 and leans on the volume sign.
func loopHandedness(polys [][]math.Point2) float64 {
	if len(polys) == 0 {
		return 1
	}
	if signedArea2D(polys[0]) < 0 {
		return -1
	}
	return 1
}

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
			frac := cellTrimFraction(f.polys, u0, u0+du, v0, v0+dv)
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
