// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestDraftFacesNeutralPivotsOnNeutralPlane is the #1801 acceptance for the neutral plane: drafting
// the +X face of a box about a neutral plane at z=1 pivots the face on the line x=2,z=1 — so a point
// on that line stays fixed, whereas the implicit (no-neutral) hinge would pin the bottom edge (z=0).
// The result is a valid tapered solid.
func TestDraftFacesNeutralPivotsOnNeutralPlane(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	side := plusXFaceKey(t, box)
	neutral, _ := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1)) // z=1, the mid-height parting plane

	res, err := blend.DraftFacesNeutral(box, [][]byte{side}, math.V3(0, 0, 1), &neutral, stdmath.Atan(0.2))
	if err != nil {
		t.Fatalf("draft with neutral plane: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("neutral-plane draft is not a valid solid: %+v", r.Issues)
	}
	drafted := mostPlusXFace(res)
	if drafted == nil {
		t.Fatal("no drafted +X face in the result")
	}
	pl, ok := drafted.Geometry().(geom.Plane)
	if !ok {
		t.Fatalf("drafted face is %T, want a plane", drafted.Geometry())
	}
	// A point on the neutral∩face hinge (x=2, z=1) must remain on the drafted plane (unmoved pivot).
	if d := geom.SignedDistanceToPlane(pl, math.P3(2, 1, 1)); stdmath.Abs(d) > 1e-6 {
		t.Errorf("hinge point (2,1,1) is %g off the drafted plane — neutral plane is not the fixed pivot", d)
	}
}

// TestDraftFacesNeutralVolumeMatchesOCCT grounds the neutral-plane draft against OCCT
// BRepOffsetAPI_DraftAngle (the oracle rule): drafting the +X face of a 2×2×2 box by 10°
// about a neutral plane at z=0.5 removes/adds a wedge of exactly ½·|z-centroid asymmetry|.
// OCCT reports 7.6473460386 for this case; the box is 8, so the wedge magnitude is
// 0.352662. Our sign convention may lean the face the other way, so we compare |Δvol|.
// Analytic: ∫₀²2·(z−0.5)·tan10° dz = tan10° = 0.1763269, ×2 = 0.3526538 (OCCT 0.3526540).
func TestDraftFacesNeutralVolumeMatchesOCCT(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	side := plusXFaceKey(t, box)
	neutral, _ := geom.NewPlane(math.P3(0, 0, 0.5), math.V3(0, 0, 1))

	res, err := blend.DraftFacesNeutral(box, [][]byte{side}, math.V3(0, 0, 1), &neutral, 10*stdmath.Pi/180)
	if err != nil {
		t.Fatalf("draft with neutral plane: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("neutral-plane draft is not a valid solid: %+v", r.Issues)
	}
	vol := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	const occtWedge = 0.3526540 // 8 − 7.6473460386, from BRepOffsetAPI_DraftAngle
	if got := stdmath.Abs(vol - 8); stdmath.Abs(got-occtWedge) > 1e-4 {
		t.Errorf("neutral-draft wedge = %g (vol %g), OCCT oracle wants %g", got, vol, occtWedge)
	}
}

// plusXFaceKey returns the reference key of the box face whose outward normal is +X.
func plusXFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if n := f.Geometry().NormalAt(0, 0); stdmath.Abs(float64(n.X)-1) < 1e-9 {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no +X face")
	return nil
}

// mostPlusXFace returns the planar face whose normal has the largest +X component (the drafted wall).
func mostPlusXFace(b *topo.Body) *topo.Face {
	var best *topo.Face
	bestX := 0.5
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			continue
		}
		if x := float64(f.Geometry().NormalAt(0, 0).X); x > bestX {
			bestX, best = x, f
		}
	}
	return best
}
