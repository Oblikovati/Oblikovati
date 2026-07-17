// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"testing"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
)

// TestExtentOfBuildsAToTargetNotALength pins that a "To <face>" extrude is built from its TARGET and
// never from its stated length (#30).
//
// Such an extrude's Distance is a stale leftover, so building it as a depth puts the feature wherever
// that number points. BigChunkyPlate has ELEVEN To extrudes; one alone — a join on the z=1.8 plane
// whose stale length read 2.6, running to z=-0.8 — made the plate 3.80 cm thick against Inventor's
// own 3.00 and cost it every one of its 47 features to the body gate. Built from the target (z=0.2,
// i.e. 1.6 away) the plate is exactly 3.00 and the features survive.
func TestExtentOfBuildsAToTargetNotALength(t *testing.T) {
	// The real shape of BigChunkyPlate's ex31: a sketch on z=1.8 facing -Z, target z=0.2.
	ex := ipt.Extrude{
		Distance: 2.6, // stale
		Dir:      [3]float64{0, 0, -1},
		DirOK:    true,
		ToPlane: ipt.SketchPlacement{
			Origin: [3]float64{0, 4.5, 0.2},
			XAxis:  [3]float64{-1, 0, 0},
			YAxis:  [3]float64{0, 1, 0},
		},
		ToPlaneOK: true,
	}
	e := extentOf(ex)
	if e.Type != feature.ToFaceExtent {
		t.Fatalf("extent type = %v, want ToFaceExtent — a To extrude must not be built as a length", e.Type)
	}
	if e.Distance != nil {
		t.Errorf("a To extent carries a Distance (%v) — its stated length is stale and must not be used", e.Distance())
	}
	if e.ToPlane == nil {
		t.Fatal("a To extent has no target plane")
	}
	o := e.ToPlane.Plane().Origin()
	if float64(o.Z) != 0.2 {
		t.Errorf("target plane origin Z = %v, want 0.2", o.Z)
	}
	// The target's normal must come back as xAxis x yAxis, the same convention as a sketch placement.
	if n := e.ToPlane.Plane().Normal().AsVector(); float64(n.Z) != -1 {
		t.Errorf("target plane normal = %v, want (0,0,-1)", n)
	}
}

// TestExtentOfKeepsTheLengthWhenNoTargetDecodes pins the fallback: an extrude with no To target keeps
// its stated length, so a file this layout cannot read is built as before rather than lost.
func TestExtentOfKeepsTheLengthWhenNoTargetDecodes(t *testing.T) {
	e := extentOf(ipt.Extrude{Distance: 2.6, Dir: [3]float64{0, 0, -1}, DirOK: true})
	if e.Type != feature.DistanceExtent {
		t.Errorf("extent type = %v, want DistanceExtent", e.Type)
	}
	if e.Distance == nil || e.Distance() != 2.6 {
		t.Errorf("a plain extrude lost its length")
	}
}
