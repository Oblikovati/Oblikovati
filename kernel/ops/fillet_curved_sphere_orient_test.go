// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// FR5 — targeted coverage for the sphere-host loop-orientation seed (orientForSphereHost and helpers). The
// D5/E4 corpus exercises only the taken path; these build the sub-hemisphere / >hemisphere / degenerate
// cases directly on a real geom.Sphere so compactSpherePole, loopTurnsNegative and reverseFilletFace are
// all driven with exact geometry.

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

func TestCompactSpherePole(t *testing.T) {
	sph := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 10}
	pole, ok := compactSpherePole(sph, spherePatchLoop(sph, 30*stdmath.Pi/180, 6))
	if !ok || float64(pole.Z) < 0.99 {
		t.Fatalf("compactSpherePole(30° patch) = (%v,%v), want pole≈+Z, compact", pole, ok)
	}
	// a lopsided loop: a tight +Z cluster whose mean points ≈+Z, plus one vertex past the equator — that
	// vertex sits >90° from the mean, so the patch is NOT sub-hemispheric.
	lopsided := append(spherePatchLoop(sph, 5*stdmath.Pi/180, 8), spherePatchLoop(sph, 150*stdmath.Pi/180, 1)...)
	if _, ok := compactSpherePole(sph, lopsided); ok {
		t.Fatal("compactSpherePole(lopsided patch) = compact, want false (a vertex >90° from the mean)")
	}
	antipodal := []math.Point3{math.P3(10, 0, 0), math.P3(-10, 0, 0), math.P3(10, 0, 0), math.P3(-10, 0, 0)}
	if _, ok := compactSpherePole(sph, antipodal); ok {
		t.Fatal("compactSpherePole(zero-sum dirs) = compact, want false (no mean direction)")
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

// TestOrientForSphereHostSeedsHost drives the whole seed: a sub-hemisphere host sphere in `all` is moved to
// index 0 (so orientFilletShell keeps its sense). A weld with NO sphere host is returned untouched.
func TestOrientForSphereHostSeedsHost(t *testing.T) {
	sph := geom.Sphere{Center: math.P3(0, 0, 0), Radius: 150}
	host := filletFace{surface: sph, loops: []filletLoop{{pts: spherePatchLoop(sph, 20*stdmath.Pi/180, 8)}}}
	arm := filletFace{surface: geom.Torus{}, loops: []filletLoop{{pts: []math.Point3{math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1)}}}}
	out := orientForSphereHost([]filletFace{arm, host}, []filletFace{host})
	if _, ok := out[0].surface.(geom.Sphere); !ok {
		t.Fatalf("orientForSphereHost did not seed faces[0] from the host sphere: got %T", out[0].surface)
	}
	noHost := []filletFace{arm}
	if got := orientForSphereHost(noHost, nil); len(got) != 1 || got[0].surface != arm.surface {
		t.Fatal("orientForSphereHost with no sphere host must return the weld untouched")
	}
}
