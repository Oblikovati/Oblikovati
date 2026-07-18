// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Targeted coverage for the sphere-host loop-orientation seed (orientForSphereHost and helpers). The
// D5/E4/D9 corpus exercises the taken path; these drive sphereLoopVectorArea, loopTurnsNegative and
// reverseFilletFace directly with exact geometry, plus the seed/do-no-harm branches of orientForSphereHost.

// spherePatchLoop returns a ring of points on sphere sph at polar angle `polar` (radians from +Z), CCW in φ.
func spherePatchLoop(sph geom.Sphere, polar float64, n int) []math.Point3 {
	pts := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		phi := 2 * stdmath.Pi * float64(i) / float64(n)
		dir := math.V3(math.Scalar(stdmath.Sin(polar)*stdmath.Cos(phi)),
			math.Scalar(stdmath.Sin(polar)*stdmath.Sin(phi)), math.Scalar(stdmath.Cos(polar)))
		pts[i] = sph.Center.TranslateBy(dir.Scale(math.Scalar(sph.Radius)))
	}
	return pts
}

// TestSphereLoopVectorArea proves the material-interior pole helper: a loop wound CCW-seen-from-outside
// around a +Z cap has a vector area (∫∫ r̂ dA) pointing at +Z (the cap centroid), and reversing the loop
// flips it to −Z — the property that lets orientForSphereHost read the zone interior off the original
// face's OUTWARD loop for both a sub- and a >hemisphere host.
func TestSphereLoopVectorArea(t *testing.T) {
	sph := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 10}
	loop := spherePatchLoop(sph, 30*stdmath.Pi/180, 12) // CCW in φ = CCW-seen-from-outside about +Z
	a, err := math.UnitVector3FromVector(sphereLoopVectorArea(sph, loop))
	if err != nil || float64(a.AsVector().Z) < 0.99 {
		t.Fatalf("sphereLoopVectorArea(+Z cap) = (%v,%v), want ≈+Z (the cap centroid)", a, err)
	}
	r, _ := math.UnitVector3FromVector(sphereLoopVectorArea(sph, reverse3(loop)))
	if float64(r.AsVector().Z) > -0.99 {
		t.Fatalf("sphereLoopVectorArea(reversed) = %v, want ≈−Z (the complement side)", r)
	}
}

func TestLoopTurnsNegativeFlipsWithOrder(t *testing.T) {
	sph := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 10}
	pole := math.V3(0, 0, 1)
	loop := spherePatchLoop(sph, 30*stdmath.Pi/180, 6)
	rev := reverse3(loop)
	if loopTurnsNegative(sph, pole, loop) == loopTurnsNegative(sph, pole, rev) {
		t.Fatal("loopTurnsNegative gave the same sign for a loop and its reverse; the winding test is broken")
	}
}

func TestReverseFilletFacePreservesMetadata(t *testing.T) {
	sph := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 10}
	pts := spherePatchLoop(sph, 30*stdmath.Pi/180, 4)
	f := filletFace{surface: sph, loops: []filletLoop{{pts: pts}}}
	rev := reverseFilletFace(f)
	if rev.surface != f.surface || len(rev.loops) != 1 {
		t.Fatalf("reverseFilletFace changed surface/loop count: %v", rev)
	}
	// reverseFilletLoop keeps the start vertex fixed and reverses the remainder, so pts[1] becomes the old last.
	if rev.loops[0].pts[1] != pts[len(pts)-1] {
		t.Fatalf("reverseFilletFace did not reverse the loop: pts[1]=%v want old-last=%v", rev.loops[0].pts[1], pts[len(pts)-1])
	}
}

// TestOrientForSphereHostSeedsHost drives the seed against the real D9 body (which carries the original
// R=150 host sphere the material pole is read from): a host sphere in `all` is moved to index 0 (so
// orientFilletShell keeps its sense). A weld with NO sphere host, and a nil body, are returned untouched.
func TestOrientForSphereHostSeedsHost(t *testing.T) {
	body := corpusFixture(t, "simple/D9.step")
	sph, ok := firstSphereSurface(body)
	if !ok {
		t.Fatal("D9 fixture carries no sphere face")
	}
	host := filletFace{surface: sph, loops: []filletLoop{{pts: spherePatchLoop(sph, 20*stdmath.Pi/180, 8)}}}
	arm := filletFace{surface: geom.Torus{}, loops: []filletLoop{{pts: []math.Point3{math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1)}}}}
	out := orientForSphereHost(body, []filletFace{arm, host}, []filletFace{host})
	if _, ok := out[0].surface.(geom.Sphere); !ok {
		t.Fatalf("orientForSphereHost did not seed faces[0] from the host sphere: got %T", out[0].surface)
	}
	if got := orientForSphereHost(body, []filletFace{arm}, nil); len(got) != 1 || got[0].surface != arm.surface {
		t.Fatal("orientForSphereHost with no sphere host must return the weld untouched")
	}
	if got := orientForSphereHost(nil, []filletFace{arm, host}, []filletFace{host}); len(got) != 2 || got[0].surface != arm.surface {
		t.Fatal("orientForSphereHost with a nil body must return the weld untouched (do-no-harm)")
	}
}

// firstSphereSurface returns the first geom.Sphere face surface on body, or ok=false.
func firstSphereSurface(body *topo.Body) (geom.Sphere, bool) {
	for _, f := range body.Faces() {
		if s, ok := f.Geometry().(geom.Sphere); ok {
			return s, true
		}
	}
	return geom.Sphere{}, false
}
