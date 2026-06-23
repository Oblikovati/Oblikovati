// SPDX-License-Identifier: GPL-2.0-only

package compdef_test

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// projectedOf returns the first projected point and curve in a sketch (nil when absent).
func projectedOf(sk *sketch.Sketch) (*sketch.ProjectedPoint, *sketch.ProjectedCurve) {
	var pp *sketch.ProjectedPoint
	var pc *sketch.ProjectedCurve
	for _, e := range sk.Entities() {
		switch v := e.(type) {
		case *sketch.ProjectedPoint:
			pp = v
		case *sketch.ProjectedCurve:
			pc = v
		}
	}
	return pp, pc
}

// TestProjectedGeometryRoundTrips: a sketch with a projected origin centre point, a projected
// work plane (→ reference line), and a coincident constraint to the projected anchor now
// serializes (it used to fail with "no codec" — the save-breaking #1268 regression) and restores
// re-linked/associative with its geometry, ids and constraint intact.
func TestProjectedGeometryRoundTrips(t *testing.T) {
	def := compdef.NewPartComponentDefinition()
	sk := def.Sketches().Add(sketch.XYPlane())
	pp := sk.ProjectPoint(compdef.NewWorkPointRefSource(def, feature.OriginCenter))              // → (0,0)
	sk.ProjectCurve(compdef.NewWorkPlaneRefSource(def, feature.OriginYZPlane, sketch.XYPlane())) // YZ∩XY = Y axis line
	free := sk.Points().Add(math.P2(3, 3))
	sk.GeometricConstraints().AddCoincident(free, pp.Anchor())
	def.Recompute()

	model, err := def.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe must not fail on projected geometry: %v", err)
	}

	got := compdef.NewPartComponentDefinition()
	if err := got.ApplyRecipe(model); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	sk2 := got.Sketches().Item(0)
	pp2, pc2 := projectedOf(sk2)
	if pp2 == nil || pc2 == nil {
		t.Fatalf("projected geometry lost on reload: point=%v curve=%v", pp2, pc2)
	}
	if !pp2.Position().IsEqualTo(math.P2(0, 0), 1e-9) {
		t.Errorf("restored projected point at %v, want (0,0)", pp2.Position())
	}
	if !pp2.Linked() || pp2.SourceID() != string(feature.OriginCenter) {
		t.Errorf("projected point not re-linked to its source: linked=%v src=%q", pp2.Linked(), pp2.SourceID())
	}
	if !pc2.Linked() {
		t.Error("projected curve should be re-linked (associative) after restore")
	}
	if len(sk2.Constraints()) == 0 {
		t.Error("the coincident constraint to the projected anchor was lost on reload")
	}
}
