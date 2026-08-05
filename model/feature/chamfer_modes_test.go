// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// box2 builds a 2×2×2 box on the XY plane and returns it with the reference key of its
// vertical edge at the +X+Y corner (2,2).
func box2(t *testing.T) (*topo.Body, []byte) {
	t.Helper()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	for _, e := range box.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == b.X && a.Y == b.Y && stdmath.Abs(a.X-2) < 1e-9 && stdmath.Abs(a.Y-2) < 1e-9 {
			return box, e.ReferenceKey()
		}
	}
	t.Fatal("no vertical edge at corner (2,2)")
	return nil, nil
}

// cornerSetbacks returns the distances from the chamfered corner (2,2,0) to the chamfer
// face's setback vertices on the bottom (z=0), sorted — for an asymmetric chamfer these are
// the two different face setbacks.
func cornerSetbacks(body *topo.Body) []float64 {
	var ds []float64
	for _, v := range body.Vertices() {
		p := v.Point()
		if stdmath.Abs(p.Z) > 1e-6 {
			continue
		}
		if d := p.DistanceTo(math.P3(2, 2, 0)); d > 0.05 && d < 1.5 {
			ds = append(ds, d)
		}
	}
	sort.Float64s(ds)
	return ds
}

// TestChamferTwoDistancesAsymmetric chamfers a box edge with unequal setbacks and verifies
// the volume (½·d1·d2·L wedge) and that the two face setbacks are actually d1 and d2 (M20-F03).
func TestChamferTwoDistancesAsymmetric(t *testing.T) {
	box, edge := box2(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	ch := NewDressUpFeatures(fs).AddChamferDef(&ChamferDefinition{
		EdgeKeys: [][]byte{edge}, Distance: func() float64 { return 0.3 },
		Distance2: func() float64 { return 0.6 }, Type: types.ChamferTwoDistances, FlatCorners: true,
	})
	fs.Recompute()

	if !ch.Health().OK() {
		t.Fatalf("two-distance chamfer sick: %+v", ch.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("two-distance chamfer not a valid solid: %+v", r)
	}
	want := 8 - 0.5*0.3*0.6*2 // box 8 − asymmetric wedge ½·d1·d2·length
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("two-distance chamfer volume = %g, want %g", got, want)
	}
	ds := cornerSetbacks(res)
	if len(ds) != 2 || stdmath.Abs(ds[0]-0.3) > 1e-6 || stdmath.Abs(ds[1]-0.6) > 1e-6 {
		t.Errorf("corner setbacks = %v, want [0.3 0.6] (asymmetric)", ds)
	}
}

// TestChamferDistanceAngle chamfers with a distance + angle; the second setback is derived as
// d·tanθ, so the removed wedge volume reflects the angle (M20-F03).
func TestChamferDistanceAngle(t *testing.T) {
	box, edge := box2(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	const d, angle = 0.4, stdmath.Pi / 6 // 30° ⇒ d2 = 0.4·tan30°
	ch := NewDressUpFeatures(fs).AddChamferDef(&ChamferDefinition{
		EdgeKeys: [][]byte{edge}, Distance: func() float64 { return d },
		Angle: func() float64 { return angle }, Type: types.ChamferDistanceAndAngle, FlatCorners: true,
	})
	fs.Recompute()

	if !ch.Health().OK() {
		t.Fatalf("distance-angle chamfer sick: %+v", ch.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("distance-angle chamfer not a valid solid: %+v", r)
	}
	d2 := d * stdmath.Tan(angle)
	want := 8 - 0.5*d*d2*2
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("distance-angle chamfer volume = %g, want %g (d2=%g)", got, want, d2)
	}
}

// TestChamferTwoDistancesRoundTrip checks the mode + second distance survive an .obk round
// trip (extrude source so the program serializes).
func TestChamferTwoDistancesRoundTrip(t *testing.T) {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	NewDressUpFeatures(fs).AddChamferDef(&ChamferDefinition{
		EdgeKeys: [][]byte{[]byte("e0")}, Distance: func() float64 { return 0.3 },
		Distance2: func() float64 { return 0.6 }, Type: types.ChamferTwoDistances, FlatCorners: true,
	})

	data, err := fs.MarshalRecipe(oneSketch{sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{sk}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(1).Definition().(*ChamferFeature).Definition()
	if def.Type != types.ChamferTwoDistances {
		t.Errorf("restored chamfer type = %v, want twoDistances", def.Type)
	}
	if def.Distance() != 0.3 || def.Distance2() != 0.6 {
		t.Errorf("restored setbacks = (%g, %g), want (0.3, 0.6)", def.Distance(), def.Distance2())
	}
}
