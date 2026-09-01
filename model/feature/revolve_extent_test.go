// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Revolve extents that terminate on model geometry (#1860). Every case revolves the same 2×2 square
// at x∈[2,4] about Y — the washer the direction tests use — so the resolved stop angle is readable
// straight off the volume: a sweep of θ holds θ/2π of the full 24π washer.

// fullWasher is the volume the 2×2 square at x∈[2,4] sweeps in a complete turn: π(4²−2²)·2.
const fullWasher = stdmath.Pi * (4*4 - 2*2) * 2

// radialPlane returns a work plane through the origin containing the Y axis, whose two half-planes
// sit at ±deg from +X measured about +Y — a terminator a revolve about Y can legally stop on.
func radialPlane(t *testing.T, deg float64) *WorkPlane {
	t.Helper()
	// The stop direction w is a×n, so the plane's normal is the stop direction turned a quarter
	// turn about the axis: n = (sin, 0, cos) puts w at (cos, 0, −sin), i.e. deg about +Y from +X.
	rad := deg * stdmath.Pi / 180
	xAxis, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	yAxis, _ := math.UnitVector3FromVector(math.V3(-stdmath.Cos(rad), 0, stdmath.Sin(rad)))
	pl, err := sketch.NewPlane(math.P3(0, 0, 0), xAxis, yAxis)
	if err != nil {
		t.Fatalf("radialPlane(%g°): %v", deg, err)
	}
	return NewFixedWorkPlane(pl)
}

// revolveWithExtent builds the washer revolve under the given extent and returns its properties.
// A nil result means the feature reported the extent as sick, which several cases assert.
func revolveWithExtent(t *testing.T, set func(*RevolveDefinition)) *ops.GeometryProperties {
	t.Helper()
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(offsetSquareSketch(2, 2), 0, yWorkAxis(), angleConst(stdmath.Pi/2), ops.NewBody)
	set(pf.Definition().(*RevolveFeature).Definition())
	fs.Recompute()
	if !pf.Health().OK() {
		return nil
	}
	// A terminator-swept wedge is now an ANALYTIC sector (partial cylinder walls + planar caps), so
	// its volume is EXACT and only the display-grade mesh under-reports it. Measure at PropertyQuality
	// — the codebase's grade for reported property values (tessellate.go) — not the display default,
	// which biases an analytic curved volume ~1.5% low and would fail the 1% wedge gate below.
	props := query.BodyGeometryProperties(fs.Result()[0], ops.PropertyQuality())
	return &props
}

// TestRevolveToFaceStopsAtTheNearerHalfPlane is the point of the extent: the sweep runs until it
// reaches the named face, so the angle comes from the model instead of being hand-computed into the
// args. A plane has TWO half-planes and the sweep stops at whichever it reaches first — the 200°
// case is the one that proves it, since its other face stands at 20° and is what the sweep meets.
func TestRevolveToFaceStopsAtTheNearerHalfPlane(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ drawnAt, stopsAt float64 }{
		{90, 90}, {135, 135}, {200, 20},
	} {
		got := revolveWithExtent(t, func(d *RevolveDefinition) {
			d.Extent, d.ToPlane = ToFaceExtent, radialPlane(t, tc.drawnAt)
		})
		if got == nil {
			t.Fatalf("to-face revolve onto a %g° plane went sick", tc.drawnAt)
		}
		if want := fullWasher * tc.stopsAt / 360; relErr(got.Volume, want) > 0.01 {
			t.Errorf("a terminator drawn at %g° swept %g cm³, want ≈%g (a %g° wedge)",
				tc.drawnAt, got.Volume, want, tc.stopsAt)
		}
	}
}

// TestRevolveToFaceOnTheProfilesOwnPlaneIsAFullTurn: a terminator the profile already lies in is
// reached at zero, which would build nothing. A full revolution is the only reading that produces a
// solid, so the extent takes it rather than emitting an empty body.
func TestRevolveToFaceOnTheProfilesOwnPlaneIsAFullTurn(t *testing.T) {
	t.Parallel()
	got := revolveWithExtent(t, func(d *RevolveDefinition) {
		d.Extent, d.ToPlane = ToFaceExtent, radialPlane(t, 0)
	})
	if got == nil {
		t.Fatal("to-face revolve onto the profile's own plane went sick; want a full revolution")
	}
	if relErr(got.Volume, fullWasher) > 0.01 {
		t.Errorf("volume = %g, want the full washer ≈%g", got.Volume, fullWasher)
	}
}

// TestRevolveToFaceRefusesATargetThatMissesTheAxis is the honest-limitation guard: a terminator
// that does not contain the axis meets the sweep at a different angle for every profile point, so
// there is no one stop angle. Approximating it would silently build the wrong wedge, so the feature
// declines — both for a target square to the axis and for one parallel to it but offset.
func TestRevolveToFaceRefusesATargetThatMissesTheAxis(t *testing.T) {
	t.Parallel()
	for name, target := range map[string]*WorkPlane{
		"square to the axis": NewFixedWorkPlane(sketch.XZPlane()),
		"offset off it":      offsetRadialPlane(t, 1),
	} {
		if got := revolveWithExtent(t, func(d *RevolveDefinition) {
			d.Extent, d.ToPlane = ToFaceExtent, target
		}); got != nil {
			t.Errorf("a to-face target %s built a %g cm³ solid; it must be refused", name, got.Volume)
		}
	}
}

// offsetRadialPlane returns a plane PARALLEL to the axis but standing off it by x — the near miss
// that orientation alone cannot catch.
func offsetRadialPlane(t *testing.T, x float64) *WorkPlane {
	t.Helper()
	xAxis, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	yAxis, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	pl, err := sketch.NewPlane(math.P3(math.Scalar(x), 0, 0), xAxis, yAxis)
	if err != nil {
		t.Fatalf("offsetRadialPlane(%g): %v", x, err)
	}
	return NewFixedWorkPlane(pl)
}

// TestRevolveFromToBoundsTheWedgeBothWays: from-to sweeps backwards to the "from" terminator and
// forwards to the "to" one, so the wedge spans both and always contains the profile that generated
// it. From 90° back to 45° forward is a 135° wedge straddling the profile plane.
func TestRevolveFromToBoundsTheWedgeBothWays(t *testing.T) {
	t.Parallel()
	got := revolveWithExtent(t, func(d *RevolveDefinition) {
		d.Extent = FromToExtent
		d.FromPlane, d.ToPlane = radialPlane(t, 90), radialPlane(t, 45)
	})
	if got == nil {
		t.Fatal("from-to revolve went sick; want a 135° wedge")
	}
	if want := fullWasher * 135 / 360; relErr(got.Volume, want) > 0.01 {
		t.Errorf("from-to volume = %g, want ≈%g (90° back + 45° forward)", got.Volume, want)
	}
}

// TestRevolveToNextStopsAtTheFirstMaterial: with no named terminator the stop angle has to be
// found, so the profile's own points are marched round their circular paths. The wall sits shallow
// enough that BOTH the outer (r=4) and inner (r=2) corners eventually reach it, at 22° and 49°
// respectively — so the case pins that the FIRST arrival stops the feature, not the last.
func TestRevolveToNextStopsAtTheFirstMaterial(t *testing.T) {
	t.Parallel()
	wall := wallAcrossTheSweep(1.5)
	base := modelPolygon(mustProfile(t, offsetSquareSketch(2, 2)), sketch.XYPlane())
	got, err := revolveToNextTurn(base, yWorkAxis(), []*topo.Body{wall}, senseOf(PositiveDir))
	if err != nil {
		t.Fatalf("to-next: %v", err)
	}
	want := stdmath.Asin(1.5 / 4) // the outer corner arrives first; the inner one at asin(1.5/2)
	if stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("to-next turn = %g rad, want ≈%g (the OUTER corner's arrival, not the inner one at %g)",
			got, want, stdmath.Asin(1.5/2))
	}
}

// TestRevolveToNextNeedsMaterialAhead: nothing to stop on is a lost input the feature reports, not
// a silent full revolution.
func TestRevolveToNextNeedsMaterialAhead(t *testing.T) {
	t.Parallel()
	base := modelPolygon(mustProfile(t, offsetSquareSketch(2, 2)), sketch.XYPlane())
	if _, err := revolveToNextTurn(base, yWorkAxis(), nil, senseOf(PositiveDir)); err == nil {
		t.Error("to-next with no bodies returned a span; it must report that nothing lies ahead")
	}
}

// TestRevolveExtentRoundTrips: a saved part must reopen with the same termination. Without the
// recipe carrying it, a to-face revolve would reload as a plain angle revolve — the sweep frozen at
// whatever the model happened to resolve last, exactly the parametric loss the extent removes.
func TestRevolveExtentRoundTrips(t *testing.T) {
	t.Parallel()
	sk, work := offsetSquareSketch(2, 2), NewWorkGeometry()
	yz, err := work.WorkPlaneByRef(WorkRef("origin/plane/yz"))
	if err != nil {
		t.Fatalf("origin YZ plane: %v", err)
	}
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).Add(sk, 0, yWorkAxis(), angleConst(stdmath.Pi/2), ops.NewBody)
	def := pf.Definition().(*RevolveFeature).Definition()
	def.Extent, def.ToPlane, def.FromPlane = FromToExtent, yz, yz

	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if got := data[0].Revolve.Extent; got != "from-to" {
		t.Fatalf("recipe extent = %q, want \"from-to\"", got)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, work); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	back := fresh.Item(0).Definition().(*RevolveFeature).Definition()
	if back.Extent != FromToExtent || back.ToPlane != yz || back.FromPlane != yz {
		t.Errorf("restored extent = %v with to=%v from=%v, want from-to bounded by the YZ plane both ways",
			back.Extent, back.ToPlane, back.FromPlane)
	}
}

// TestAngleRevolveRecipeIsUnchanged: the angle extent is the default, so it must write nothing —
// an ordinary revolve's saved recipe is byte-identical to what it was before #1860.
func TestAngleRevolveRecipeIsUnchanged(t *testing.T) {
	t.Parallel()
	sk := offsetSquareSketch(2, 2)
	fs := NewPartFeatures(nil)
	NewRevolveFeatures(fs).Add(sk, 0, yWorkAxis(), angleConst(stdmath.Pi/2), ops.NewBody)
	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].Revolve; d.Extent != "" || d.ToPlane != "" || d.FromPlane != "" {
		t.Errorf("angle revolve wrote extent=%q to=%q from=%q; all three must stay absent",
			d.Extent, d.ToPlane, d.FromPlane)
	}
}

// wallAcrossTheSweep is a slab spanning the washer's radii at |z| = depth, so a sweep about Y runs
// into it. It is deliberately wide in x and y and thin in z, so the arrival angle is decided by the
// sweeping point's radius alone.
func wallAcrossTheSweep(depth float64) *topo.Body {
	b := subd.ToBody(subd.Box(10, 4, 0.5), "wall")
	keep := func(l topo.Lineage) topo.Lineage { return l }
	moved, err := transform.TransformBody(b, math.Translation4(math.V3(-5, -1, -depth-0.5)), keep)
	if err != nil {
		panic(err) // a translation of a box cannot fail; a panic here is a broken fixture, not a case
	}
	return moved
}

// mustProfile resolves the sketch's single closed region, failing the test if it has none.
func mustProfile(t *testing.T, s *sketch.Sketch) *sketch.Profile {
	t.Helper()
	prof, err := resolveSingleProfile(s, 0, "revolve")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	return prof
}
