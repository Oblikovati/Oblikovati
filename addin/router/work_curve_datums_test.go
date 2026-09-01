// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// unitBoxPartSession returns a router + session whose active part carries a unit box (0,0,0)-(1,1,1),
// so edge-based work-point kinds have real B-rep edges to bind their geometric references to.
func unitBoxPartSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	block, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "widget")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	feature.NewBaseFeatures(part.Features()).AddBase(block)
	part.Recompute()
	return r, s
}

func edgeGeomRef(mid, dir [3]float64) string {
	return types.GeometricEdgeRef{Midpoint: mid, Direction: dir}.Ref()
}

// TestEdgeMidpointResolvesOverWire proves the wire path binds an edge geometric reference: with a
// box in the part, an edge-midpoint point resolves to the edge midpoint. Regression for the
// ParseWorkRef mangling that had every edge datum go unhealthy over the wire (#1840, #1842).
func TestEdgeMidpointResolvesOverWire(t *testing.T) {
	t.Parallel()
	r, s := unitBoxPartSession(t)
	ref := edgeGeomRef([3]float64{0.5, 0, 1}, [3]float64{1, 0, 0}) // top edge (0,0,1)-(1,0,1)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"kind":"edge-midpoint","refs":["`+ref+`"]}`, &res)
	if !res.Healthy {
		t.Fatalf("edge-midpoint should resolve against the box edge over the wire: %+v", res)
	}
}

// TestCurveEntityResolvesOverWire: a box edge pierces the YZ origin plane, and the curve-and-entity
// point resolves over the wire (#1842) — depends on the edge ref surviving ParseWorkRef.
func TestCurveEntityResolvesOverWire(t *testing.T) {
	t.Parallel()
	r, s := unitBoxPartSession(t)
	ref := edgeGeomRef([3]float64{0.5, 0, 1}, [3]float64{1, 0, 0}) // top edge (0,0,1)-(1,0,1), meets x=0 at (0,0,1)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"kind":"curve-and-entity","refs":["`+ref+`","origin/plane/yz"],"proximity":[0,0,1]}`, &res)
	if !res.Healthy {
		t.Fatalf("curve-and-entity should resolve over the wire: %+v", res)
	}
}

// TestCentroidResolvesOverWire: the centroid of two box edges resolves over the wire (#1842).
func TestCentroidResolvesOverWire(t *testing.T) {
	t.Parallel()
	r, s := unitBoxPartSession(t)
	e0 := edgeGeomRef([3]float64{0.5, 0, 0}, [3]float64{1, 0, 0}) // (0,0,0)-(1,0,0)
	e1 := edgeGeomRef([3]float64{1, 0.5, 0}, [3]float64{0, 1, 0}) // (1,0,0)-(1,1,0)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"kind":"centroid","refs":["`+e0+`","`+e1+`"]}`, &res)
	if !res.Healthy {
		t.Fatalf("centroid should resolve over the wire: %+v", res)
	}
}

// TestCurveEntityCentroidBadArgs: arity/argument guards report clean errors, not panics (#1842).
func TestCurveEntityCentroidBadArgs(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	bad := []string{
		`{"kind":"curve-and-entity","refs":["edge/AAAA"]}`,                                     // needs 2 refs
		`{"kind":"curve-and-entity","refs":["edge/AAAA","origin/plane/yz"],"proximity":[0,1]}`, // bad proximity length
		`{"kind":"centroid","refs":[]}`,                                                        // needs ≥1 edge
		`{"kind":"cloud-point","refs":["a","b"],"at":[0,0,0]}`,                                 // needs exactly 1 cloud ref
	}
	for _, args := range bad {
		if _, err := r.Handle(s, "workPoints.create", []byte(args)); err == nil {
			t.Errorf("expected an error for %s", args)
		}
	}
}
