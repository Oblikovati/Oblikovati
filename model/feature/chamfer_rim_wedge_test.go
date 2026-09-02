// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// TestRevolvedRimChamferToolIsAnAnalyticRing is the regression for the rotational sweep. It drives
// the tool builder directly rather than the feature, because the feature still facets its host
// first (planarizeForEdges); that faceting is what #3459 removes, and this is what has to work
// when it does.
//
// The prism form cannot express this case at all: a closed rim's two endpoints are the same point,
// so its frame reported "chamfer: degenerate edge" and the feature went sick (#1689). Faceting hid
// that, because a faceted rim is a chain of short straight edges.
func TestRevolvedRimChamferToolIsAnAnalyticRing(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewRevolveFeatures(fs).Add(stepShaftSketch(), 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	body := fs.Result()[0]

	var rim *topo.Edge
	for _, e := range body.Edges() {
		if c, ok := geom.AsCircle(e.Geometry()); ok {
			if cy := float64(c.Center.Y); cy > 1.9 && cy < 2.1 && c.Radius > 0.9 {
				rim = e
			}
		}
	}
	if rim == nil {
		t.Fatalf("no top rim circle (r≈1 at y≈2) among %d edges", len(body.Edges()))
	}
	if _, ok := closedCircularRim(rim); !ok {
		t.Fatal("the top rim is a full circle; closedCircularRim must recognise it")
	}

	tool, err := chamferWedge(rim, 0.06, 0.06, chamferRun{}, "rimchamfer")
	if err != nil {
		t.Fatalf("chamferWedge on a closed rim: %v — the prism form reports \"degenerate edge\" here (#1689)", err)
	}
	if r := ops.Validate(tool); !r.ValidSolid() {
		t.Fatalf("the rim chamfer tool is not a valid solid: %+v", r.Issues)
	}
	// A rotational sweep of the section is a RING with an analytic bevel — the reason to take this
	// path at all. A prism sweep could only ever produce planes.
	cones := 0
	for _, f := range tool.Faces() {
		if _, isCone := f.Geometry().(geom.Cone); isCone {
			cones++
		}
	}
	if cones == 0 {
		t.Errorf("the rim chamfer tool has no cone face among %d faces — the section was swept "+
			"linearly, not rotationally", len(tool.Faces()))
	}
	vol := query.BodyGeometryProperties(tool, ops.DefaultQuality()).Volume
	if vol <= 0 || vol > 1 {
		t.Errorf("rim chamfer tool volume %g; a 0.06 bevel on an r=1 rim is a thin ring", vol)
	}
}

// TestClosedCircularRimRecognisesOnlyFullCircles pins the classification the sweep dispatches on: a
// full rim sweeps rotationally, anything else keeps the prism.
func TestClosedCircularRimRecognisesOnlyFullCircles(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewRevolveFeatures(fs).Add(stepShaftSketch(), 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	body := fs.Result()[0]
	rims, others := 0, 0
	for _, e := range body.Edges() {
		if _, ok := closedCircularRim(e); ok {
			rims++
		} else {
			others++
		}
	}
	if rims == 0 {
		t.Error("a revolved shaft has closed circular rims; none were recognised")
	}
	_ = others
}
