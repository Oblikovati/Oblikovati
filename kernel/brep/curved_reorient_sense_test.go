// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// shellSense sums the divergence-theorem volume the faces' STORED senses bound, and the outward vector
// area a closed shell owes that theorem. A correctly oriented solid gives a positive volume and a
// residual of zero; a face whose material side disagrees with its neighbours' shows up in the residual
// even when the total volume happens to look right.
func shellSense(t *testing.T, faces []curvedFace) (volume, residual, area float64) {
	t.Helper()
	var ax, ay, az float64
	for _, f := range faces {
		region := faceTrimRegion(f)
		u0, u1, v0, v1, ok := fluxDomain(f, region)
		if !ok {
			t.Fatalf("a %T face carries no usable uv domain, so this fixture cannot certify a sense", f.surface)
		}
		ff := fluxFace{cf: f, region: region, u0: u0, u1: u1, v0: v0, v1: v1, sign: 1}
		s := 1.0
		if f.reversed {
			s = -1
		}
		volume += s * faceVolumeTerm(&ff)
		a, mag := faceVectorArea(&ff)
		ax, ay, az = ax+s*float64(a.X), ay+s*float64(a.Y), az+s*float64(a.Z)
		area += mag
	}
	return volume, stdmath.Sqrt(ax*ax + ay*ay + az*az), area
}

// faceVectorArea integrates ∮ S_u×S_v over the face's trimmed domain — the outward vector area, which
// sums to ZERO over a closed shell whatever its shape — on the same coarse grid faceVolumeTerm uses. It
// also returns the unsigned area, the scale the residual is judged against.
func faceVectorArea(f *fluxFace) (math.Vector3, float64) {
	du := (f.u1 - f.u0) / volumeGridSteps
	dv := (f.v1 - f.v0) / volumeGridSteps
	sum, mag := math.V3(0, 0, 0), 0.0
	for i := range volumeGridSteps {
		for j := range volumeGridSteps {
			u0, v0 := f.u0+float64(i)*du, f.v0+float64(j)*dv
			frac := cellTrimFraction(f.region, u0, u0+du, v0, v0+dv)
			if frac == 0 {
				continue
			}
			su, sv := f.cf.surface.DerivativesAt(u0+0.5*du, v0+0.5*dv)
			n := su.Cross(sv).Scale(math.Scalar(du * dv * frac))
			sum, mag = sum.Add(n), mag+float64(n.Length())
		}
	}
	return sum, mag
}

// TestCurvedReorientTurnsTheShellOutward pins Oblikovati/Oblikovati#3504. The two-colouring makes every
// shared edge traversed oppositely, but it seeds at a face and propagates, so the sense the WHOLE shell
// takes is whichever sense that one face arrived with — flipping ten faces to agree with one is as
// consistent, and as manifold, as flipping the one.
//
// Handing curvedReorient a shell with every face inverted must therefore come back outward, not stay
// inverted-but-consistent. Nothing else in the pipeline catches this: Validate checks loop TRAVERSAL and
// the tessellator repairs its own per-face meshes, so an inside-out B-rep passed both while every
// analytic consumer read the wrong material side.
func TestCurvedReorientTurnsTheShellOutward(t *testing.T) {
	t.Parallel()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	faces := facesOfAny(cyl)
	if vol, _, _ := shellSense(t, curvedReorient(faces)); vol <= 0 {
		t.Errorf("an already-outward cylinder came back bounding %g; reorientation must leave it outward", vol)
	}
	inverted := make([]curvedFace, len(faces))
	for i, f := range faces {
		f.reversed = !f.reversed
		inverted[i] = f
	}
	if vol, _, _ := shellSense(t, inverted); vol >= 0 {
		t.Fatalf("the inverted fixture bounds %g; it must be negative for this test to mean anything", vol)
	}
	if vol, _, _ := shellSense(t, curvedReorient(inverted)); vol <= 0 {
		t.Errorf("an inside-out shell came back bounding %g; the two-colouring alone is happy with either "+
			"sense, so the global bit must be certified against the geometry", vol)
	}
}
