// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestDraftFacesNeutralPivotsOnNeutralPlane is the #1801 acceptance for the neutral plane: drafting
// the +X face of a box about a neutral plane at z=1 pivots the face on the line x=2,z=1 — so a point
// on that line stays fixed, whereas the implicit (no-neutral) hinge would pin the bottom edge (z=0).
// The result is a valid tapered solid.
func TestDraftFacesNeutralPivotsOnNeutralPlane(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	side := plusXFaceKey(t, box)
	neutral, _ := geom.NewPlane(math.P3(0, 0, 1), math.V3(0, 0, 1)) // z=1, the mid-height parting plane

	res, err := ops.DraftFacesNeutral(box, [][]byte{side}, math.V3(0, 0, 1), &neutral, stdmath.Atan(0.2))
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
