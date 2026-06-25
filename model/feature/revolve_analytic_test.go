// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
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

// TestAnalyticRevolveHasCylinderWalls proves #129 step 2: a full revolve of a rectilinear (tube)
// profile yields TRUE cylindrical faces — the bore + outer wall that thread/chamfer/fillet attach to
// — instead of a faceted prism.
func TestAnalyticRevolveHasCylinderWalls(t *testing.T) {
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

// TestArcProfileRevolveStaysFaceted pins the analytic boundary: a profile with a CURVED edge (a
// half-disc → sphere) is NOT made analytic — its sampled chords would shatter into tiny cone facets,
// worse than the faceted swept solid — so it revolves to a valid faceted sphere with no analytic
// cylinder/cone faces (the isStraightLoop guard; curved meridian edges remain a follow-up).
func TestArcProfileRevolveStaysFaceted(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	top := s.Points().Add(math.P2(0, 2))
	bot := s.Points().Add(math.P2(0, -2))
	s.Lines().Add(top, bot)                                                          // flat side on the Y axis
	s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(0, -2), math.P2(0, 2), true) // bulge +X through (2,0)

	fs := NewPartFeatures(nil, nil)
	NewRevolveFeatures(fs).Add(s, 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("faceted sphere revolve is not a valid solid: %+v", r.Issues)
	}
	if c, k := cylinderFaceCount(body), bodyConeCount(body); c != 0 || k != 0 {
		t.Fatalf("arc-profile revolve has %d cylinder + %d cone faces, want 0 (faceted sphere)", c, k)
	}
	want := 4.0 / 3.0 * stdmath.Pi * 8 // (4/3)πR³, R=2
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.03 {
		t.Errorf("faceted sphere volume = %g, want ≈%g", got, want)
	}
}

// TestAnalyticRevolveTubeBooleanCutsHalf is the revolve+boolean EXACTNESS regression: a through-all
// cut of an analytic revolve donut must (a) not explode/hang (the curved-tool guard, planarized →
// ops.Facet), AND (b) remove the right amount. The bug was a through-all extent measured from body
// VERTICES — an analytic body has only sparse seam vertices, so the slab collapsed to near-zero
// depth and barely cut. normalExtent now measures the range box, so the slab spans the body
// (Oblikovati/Oblikovati#129).
func TestAnalyticRevolveTubeBooleanCutsHalf(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, nil, ops.NewBody)

	// A symmetric through-all slab removing the donut's top half (y>1 of the y∈[0,2] section).
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
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("revolve+cut is not a valid solid: %+v", r.Issues)
	}
	// The donut is uniform in y over [0,2]; removing y>1 leaves the bottom half: 2π·R̄·A with the
	// section now 2 wide (r∈[2,4]) × 1 tall, R̄=3 ⇒ 2π·3·2 = 12π.
	want := 2 * stdmath.Pi * 3 * 2
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.03 {
		t.Fatalf("revolve+cut volume = %g, want ≈%g (12π half-donut) — extent too small?", got, want)
	}
}

// torusFaceCount tallies geom.Torus faces.
func torusFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
}

// TestCircleRevolveMakesAnalyticTorus proves the #129 curved-meridian follow-up (torus case): a single
// CIRCLE clear of the axis revolves to ONE analytic geom.Torus face — not hundreds of cone slivers — so a
// later boolean (the M2 torus half-space cuts) takes the exact analytic path on a natively-revolved torus.
func TestCircleRevolveMakesAnalyticTorus(t *testing.T) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.Circles().AddByCenterRadius(math.P2(5, 0), 2) // major 5, minor 2 about the Y axis
	fs := NewPartFeatures(nil, nil)
	NewRevolveFeatures(fs).Add(s, 0, yAxis(), nil, ops.NewBody)
	fs.Recompute()
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("revolved torus is not a valid solid: %+v", r.Issues)
	}
	if got := torusFaceCount(body); got != 1 {
		t.Fatalf("revolved circle has %d torus faces, want exactly 1 analytic torus (got %d total faces)", got, len(body.Faces()))
	}
	want := 2 * stdmath.Pi * stdmath.Pi * 5 * 2 * 2 // 40π²
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(got, want) > 0.03 {
		t.Errorf("revolved torus volume = %g, want ≈%g (40π²)", got, want)
	}
}

// TestNativeRevolveTorusHalfSpaceCutsAreExact drives the M2 torus half-space family end-to-end through
// NATIVE modelling: a circle revolved 360° (analytic torus) cut by a box. Each box clips one axis-aligned
// plane through a Y-axis torus (major 5, minor 2), exercising the perpendicular, axis-parallel single-oval
// (cap + complement), two-oval and figure-eight topologies. The result must take the EXACT analytic path —
// a handful of faces and watertight, never the faceted CSG fallback (which shatters into hundreds).
func TestNativeRevolveTorusHalfSpaceCutsAreExact(t *testing.T) {
	torus := func() *topo.Body {
		s := sketch.NewSketches().Add(sketch.XYPlane())
		s.Circles().AddByCenterRadius(math.P2(5, 0), 2)
		fs := NewPartFeatures(nil, nil)
		NewRevolveFeatures(fs).Add(s, 0, yAxis(), nil, ops.NewBody)
		fs.Recompute()
		return fs.Result()[0]
	}
	cases := []struct {
		name string
		bmin math.Point3 // box [bmin, (20,20,20)] clips one plane; the rest clears the torus
	}{
		{"perpendicular (y>=0)", math.P3(-20, 0, -20)},
		{"axis-parallel single oval (x>=6)", math.P3(6, -20, -20)},
		{"two-oval band (x>=2)", math.P3(2, -20, -20)},
		{"figure-eight (x>=3, tangent)", math.P3(3, -20, -20)},
	}
	for _, c := range cases {
		for _, op := range []struct {
			tag string
			op  ops.PartFeatureOperation
		}{{"∩", ops.Intersect}, {"−", ops.Cut}} {
			box, _ := brep.SolidBlock(c.bmin, math.P3(20, 20, 20), "box")
			res, err := ops.Boolean(op.op, torus(), box)
			if err != nil {
				t.Fatalf("%s %s: %v", c.name, op.tag, err)
			}
			if n := len(res.Faces()); n > 40 {
				t.Errorf("%s %s: %d faces — fell to faceted CSG, want the exact analytic path", c.name, op.tag, n)
			}
			if !res.IsSolid() {
				t.Errorf("%s %s: result is not a solid", c.name, op.tag)
			}
			for _, e := range res.Edges() {
				if len(e.Uses()) != 2 {
					t.Errorf("%s %s: non-manifold edge (%d uses)", c.name, op.tag, len(e.Uses()))
					break
				}
			}
		}
	}
}

// TestNativeObliqueRevolveTorusCutsAreExact completes native coverage of the M2 torus half-space family:
// a circle revolved 360° about a TILTED axis (0,0.6,0.8) — sketched on a work plane that contains that axis
// — yields an analytic OBLIQUE torus, and an axis-aligned box then cuts it at an oblique angle. The single
// oblique oval (cap + complement), two oblique ovals, and the oblique figure-eight each take the exact
// analytic path (a handful of faces, watertight), never faceted CSG.
func TestNativeObliqueRevolveTorusCutsAreExact(t *testing.T) {
	tiltedTorus := func() *topo.Body {
		pl, err := sketch.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0).AsUnit(), math.V3(0, 0.6, 0.8).AsUnit())
		if err != nil {
			t.Fatal(err)
		}
		s := sketch.NewSketches().Add(pl)
		s.Circles().AddByCenterRadius(math.P2(5, 0), 2) // 5 ⟂ to the axis → major 5, minor 2
		fs := NewPartFeatures(nil, nil)
		NewRevolveFeatures(fs).Add(s, 0, &WorkAxis{origin: math.P3(0, 0, 0), dir: math.V3(0, 0.6, 0.8).AsUnit()}, nil, ops.NewBody)
		fs.Recompute()
		return fs.Result()[0]
	}
	if torusFaceCount(tiltedTorus()) != 1 {
		t.Fatalf("tilted revolve has %d torus faces, want 1 analytic", torusFaceCount(tiltedTorus()))
	}
	cases := []struct {
		name string
		z    float64 // box z>=this cuts the tilted torus obliquely
	}{
		{"oblique single oval", 3.6},
		{"two oblique ovals", 0.5},
		{"oblique figure-eight", 1.0},
	}
	for _, c := range cases {
		for _, op := range []struct {
			tag string
			op  ops.PartFeatureOperation
		}{{"∩", ops.Intersect}, {"−", ops.Cut}} {
			box, _ := brep.SolidBlock(math.P3(-20, -20, c.z), math.P3(20, 20, 20), "box")
			res, err := ops.Boolean(op.op, tiltedTorus(), box)
			if err != nil {
				t.Fatalf("%s %s: %v", c.name, op.tag, err)
			}
			if n := len(res.Faces()); n > 40 {
				t.Errorf("%s %s: %d faces — fell to faceted CSG", c.name, op.tag, n)
			}
			if !res.IsSolid() {
				t.Errorf("%s %s: result is not a solid", c.name, op.tag)
			}
		}
	}
}

// TestRevolvedTorusExtrudeCutStaysAnalytic proves the combine fix: cutting a revolved (analytic) torus with
// an extruded box keeps the analytic torus surface — the feature boolean uses the M2 curved path rather than
// faceting the torus first. Complements TestAnalyticRevolveTubeBooleanCutsHalf, where a COMPOSITE washer
// (two cylinder walls) correctly stays on the faceted planar path.
func TestRevolvedTorusExtrudeCutStaysAnalytic(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(5, 0), 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 1))
	cl.SetCenterline(true)
	NewRevolveFeatures(fs).AddAboutCenterline(sk, 0, nil, ops.NewBody)

	// keep x>=6 (an axis-parallel single-oval cap) via a box on the x=6 plane intersected with the torus
	pl, err := sketch.NewPlane(math.P3(6, 0, 0), math.V3(0, 1, 0).AsUnit(), math.V3(0, 0, 1).AsUnit())
	if err != nil {
		t.Fatal(err)
	}
	clip := sketch.NewSketches().Add(pl)
	clip.AddRectangleByCorners(math.P2(-20, -20), math.P2(20, 20))
	NewExtrudeFeatures(fs).AddByDistanceExtent(clip, 0, ops.Intersect, func() float64 { return 14 })
	fs.Recompute()

	body := fs.Result()[0]
	if torusFaceCount(body) != 1 {
		t.Fatalf("torus extrude-cut has %d torus faces, want 1 (analytic kept, not planarized) — %d total faces",
			torusFaceCount(body), len(body.Faces()))
	}
	if n := len(body.Faces()); n > 40 {
		t.Errorf("torus extrude-cut has %d faces — fell to faceted CSG", n)
	}
}
