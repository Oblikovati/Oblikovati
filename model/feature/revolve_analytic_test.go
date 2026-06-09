// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"os"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/model/sketch"
)

// revolveTubeBody revolves a washer profile (x∈[2,4], y∈[0,2]) 360° about the Y axis into a tube.
func revolveTubeBody(t *testing.T) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil, nil)
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, nil, ops.NewBody)
	fs.Recompute()
	return fs.Result()[0]
}

func cylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// TestAnalyticRevolveHasCylinderWalls proves #129 step 2: under OBK_ANALYTIC_CURVES a full revolve
// of a rectilinear (tube) profile yields TRUE cylindrical faces — the bore + outer wall that
// thread/chamfer/fillet attach to — instead of a faceted prism.
func TestAnalyticRevolveHasCylinderWalls(t *testing.T) {
	os.Setenv("OBK_ANALYTIC_CURVES", "1")
	defer os.Unsetenv("OBK_ANALYTIC_CURVES")

	body := revolveTubeBody(t)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("analytic revolved tube is not a valid solid: %+v", r.Issues)
	}
	if got := cylinderFaceCount(body); got != 2 {
		t.Fatalf("analytic tube has %d cylinder faces, want 2 (bore + outer wall)", got)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 // 24π
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.03 {
		t.Errorf("analytic tube volume = %g, want ≈%g (24π)", got, want)
	}
}

// TestGatedRevolveStaysFaceted pins the gate-off default: the same revolve has NO analytic cylinder
// faces (the faceted prism path), so the analytic surface is opt-in until the gate is flipped.
func TestGatedRevolveStaysFaceted(t *testing.T) {
	os.Unsetenv("OBK_ANALYTIC_CURVES")
	if got := cylinderFaceCount(revolveTubeBody(t)); got != 0 {
		t.Fatalf("gate-off revolve has %d cylinder faces, want 0 (faceted)", got)
	}
}

// TestAnalyticRevolveTubeBooleanDoesNotExplode is the regression for the curved-tool explosion
// (#129, analogous to the patterned-cut blow-up): a boolean against an analytic revolve tube must
// re-facet it (planarized → ops.Facet) so brep.Boolean never sees a full periodic cylinder face and
// hangs / blows the body up to tens of thousands of edges. It asserts the operation COMPLETES into
// one bounded-size body that removed material.
//
// It deliberately does NOT assert an exact post-cut volume: booleans on a SOLID OF REVOLUTION are a
// pre-existing weak spot — the faceted revolve (gate off) itself yields an invalid/inexact cut here
// (a faceted-revolve winding bug), so revolve+boolean exactness is its own follow-up, separate from
// emitting the analytic faces thread attaches to (this step's goal).
func TestAnalyticRevolveTubeBooleanDoesNotExplode(t *testing.T) {
	os.Setenv("OBK_ANALYTIC_CURVES", "1")
	defer os.Unsetenv("OBK_ANALYTIC_CURVES")

	fs := NewPartFeatures(nil, nil)
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, nil, ops.NewBody)

	// A symmetric through-all slab removing the tube's top half (y>1).
	clip := sketch.NewSketches().Add(sketch.XYPlane())
	q0 := clip.Points().Add(math.P2(-10, 1))
	q1 := clip.Points().Add(math.P2(10, 1))
	q2 := clip.Points().Add(math.P2(10, 10))
	q3 := clip.Points().Add(math.P2(-10, 10))
	clip.Lines().Add(q0, q1)
	clip.Lines().Add(q1, q2)
	clip.Lines().Add(q2, q3)
	clip.Lines().Add(q3, q0)
	NewExtrudeFeatures(fs).AddExtrude(clip, []int{0}, ops.Cut, Extent{Type: ThroughAllExtent, Direction: SymmetricDir}, 0)

	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("revolve+cut = %d bodies, want 1", len(bodies))
	}
	body := bodies[0]
	if n := len(body.Edges()); n > 2000 {
		t.Fatalf("revolve+cut exploded to %d edges (curved tool not re-faceted before the boolean?)", n)
	}
	full := stdmath.Pi * (4*4 - 2*2) * 2
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; got >= full || got <= 0 {
		t.Fatalf("revolve+cut volume %g not in (0, %g): the clip removed no material", got, full)
	}
}
