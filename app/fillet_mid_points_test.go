// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/feature"
)

// TestFilletMidPointDefaults checks AddMidPoint lands a stop in the widest gap with its radius
// linearly interpolated between the start and end radii (#695): the first goes to the centre,
// the second splits the larger half.
func TestFilletMidPointDefaults(t *testing.T) {
	f := NewFilletTool()
	f.SetStartRadius(0.2)
	f.SetEndRadius(0.6)
	f.AddMidPoint()
	if got := f.MidPoints(); len(got) != 1 || got[0].T != 0.5 || got[0].Radius != 0.4 {
		t.Fatalf("first stop = %+v, want T=0.5 R=0.4 (centre, mid-interpolated)", got)
	}
	f.AddMidPoint() // widest gap is now [0,0.5] → 0.25
	if got := f.MidPoints(); len(got) != 2 || got[1].T != 0.25 {
		t.Fatalf("second stop T = %v, want 0.25 (midpoint of widest gap)", got)
	}
}

// TestFilletMidPointValidation mirrors the kernel's validateRadiusPoints contract: interior,
// strictly increasing, positive radius (#695).
func TestFilletMidPointValidation(t *testing.T) {
	cases := []struct {
		name string
		pts  []FilletMidPoint
		want bool
	}{
		{"empty is valid", nil, true},
		{"increasing interior", []FilletMidPoint{{0.3, 0.4}, {0.7, 0.5}}, true},
		{"unsorted but distinct is fine", []FilletMidPoint{{0.7, 0.5}, {0.3, 0.4}}, true},
		{"T at 0 boundary", []FilletMidPoint{{0, 0.4}}, false},
		{"T at 1 boundary", []FilletMidPoint{{1, 0.4}}, false},
		{"duplicate T", []FilletMidPoint{{0.5, 0.4}, {0.5, 0.5}}, false},
		{"non-positive radius", []FilletMidPoint{{0.5, 0}}, false},
	}
	for _, c := range cases {
		f := NewFilletTool()
		f.midPoints = c.pts
		if got := f.midPointsValid(); got != c.want {
			t.Errorf("%s: midPointsValid(%+v) = %v, want %v", c.name, c.pts, got, c.want)
		}
	}
}

// TestFilletMidPointEdit checks Set/Remove edit the addressed stop and ignore out-of-range.
func TestFilletMidPointEdit(t *testing.T) {
	f := NewFilletTool()
	f.midPoints = []FilletMidPoint{{0.25, 0.3}, {0.5, 0.4}, {0.75, 0.5}}
	f.SetMidPointT(1, 0.6)
	f.SetMidPointR(1, 0.45)
	f.SetMidPointT(9, 0.9) // out of range: no-op
	if got := f.MidPoints()[1]; got.T != 0.6 || got.Radius != 0.45 {
		t.Errorf("edited stop = %+v, want T=0.6 R=0.45", got)
	}
	f.RemoveMidPoint(0)
	if got := f.MidPoints(); len(got) != 2 || got[0].T != 0.6 {
		t.Errorf("after RemoveMidPoint(0), stops = %+v, want the 0.6 and 0.75 stops", got)
	}
	f.RemoveMidPoint(5) // out of range: no-op
	if len(f.MidPoints()) != 2 {
		t.Errorf("out-of-range remove changed the slice: %+v", f.MidPoints())
	}
}

// TestFilletMidPointCommit drives the variable fillet end to end with one intermediate stop and
// confirms the committed definition carries the radius points, sorted by T (#695). Values match
// the kernel's TestFilletIntermediateRadiiVolume (known buildable on a 2×2×2 block).
func TestFilletMidPointCommit(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
	f := NewFilletTool()
	s.StartTool(f)
	s.Click(1, 1)
	f.SetVariable(true)
	f.SetStartRadius(0.3)
	f.SetEndRadius(0.4)
	f.midPoints = []FilletMidPoint{{T: 0.5, Radius: 0.7}}
	if !f.CanCommit() {
		t.Fatal("variable fillet with a valid stop should be committable")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := f.AddedFeature().Definition().(*feature.FilletFeature).Definition()
	if len(def.EdgeSets) != 1 {
		t.Fatalf("committed fillet has %d edge sets, want 1", len(def.EdgeSets))
	}
	pts := def.EdgeSets[0].RadiusPoints
	if len(pts) != 1 || pts[0].T != 0.5 || pts[0].Radius() != 0.7 {
		t.Errorf("committed radius points = %+v, want one stop T=0.5 R=0.7", pts)
	}
}

// TestFilletMidPointEditSeed checks re-editing a committed variable fillet seeds the panel's
// stops from the feature's radius points (#695).
func TestFilletMidPointEditSeed(t *testing.T) {
	set := feature.FilletEdgeSet{
		EdgeKeys:     [][]byte{{1, 2, 3}},
		StartRadius:  func() float64 { return 0.3 },
		EndRadius:    func() float64 { return 0.4 },
		RadiusPoints: []feature.FilletRadiusPoint{{T: 0.5, Radius: func() float64 { return 0.7 }}},
	}
	tool := NewFilletTool()
	seedFilletSet(tool, set)
	if !tool.Variable() {
		t.Fatal("seeding a variable set should put the tool in variable mode")
	}
	if got := tool.MidPoints(); len(got) != 1 || got[0].T != 0.5 || got[0].Radius != 0.7 {
		t.Errorf("seeded stops = %+v, want one stop T=0.5 R=0.7", got)
	}
}
