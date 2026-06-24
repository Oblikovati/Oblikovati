// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// An oblique plane tilted SHALLOWER than the cone's generators (but not axis-parallel) cuts the frustum
// side in a hyperbola whose vertex is below the band; the kept side is an exact arc-band cone face
// bounded by the two hyperbola arms and the kept rim arcs, watertight (Oblikovati/Oblikovati#1375).
func TestConeSideHalfSpaceObliqueHyperbola(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)                    // α≈16.7°
	plane, err := geom.NewPlane(math.P3(2, 0, 0), math.V3(stdmath.Sqrt(1-0.2*0.2), 0, 0.2)) // φ≈11.5° < α
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("got %d cone faces, want 1", cones)
	}
	if !anyFaceHasHyperbola(res) {
		t.Error("no hyperbolic edge — the oblique cut was not imprinted as a hyperbola")
	}
}

// An oblique hyperbola whose vertex falls INSIDE the band (the arms turn before reaching the small rim,
// the oblique analogue of #1374) is not yet built and must defer cleanly so the CSG fallback covers it.
func TestConeSideHalfSpaceObliqueHyperbolaVertexInsideDefers(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(4, 0, 0), math.V3(stdmath.Sqrt(1-0.2*0.2), 0, 0.2)) // vertex apex-dist ≈ 12, in band
	if _, err := HalfSpaceCut(frustum, plane); err == nil {
		t.Error("vertex-inside oblique hyperbola should defer, got nil error")
	}
}

// A plane shallower than the generators but clear of the frustum band (its section lies on the infinite
// cone beyond the rims) keeps the side whole / empties it rather than deferring.
func TestConeSideHalfSpaceObliqueHyperbolaClears(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	keep, _ := geom.NewPlane(math.P3(40, 0, 0), math.V3(stdmath.Sqrt(1-0.2*0.2), 0, 0.2)) // far +x: whole kept
	res, err := HalfSpaceCut(frustum, keep)
	if err != nil {
		t.Fatalf("HalfSpaceCut(keep): %v", err)
	}
	if len(res.Faces()) != len(frustum.Faces()) {
		t.Errorf("clearing plane changed the face count: %d vs %d", len(res.Faces()), len(frustum.Faces()))
	}
}
