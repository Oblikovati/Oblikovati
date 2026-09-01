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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestChamferReferenceFaceDecidesWhichSetbackGoesWhere (#1888): an asymmetric chamfer is only
// meaningful once you say which face the first distance is measured on. Without a reference the
// answer came from edge.Faces() order — a topology artefact — so naming each of the edge's two
// faces in turn must SWAP where the big setback lands, and reach the same volume either way.
func TestChamferReferenceFaceDecidesWhichSetbackGoesWhere(t *testing.T) {
	t.Parallel()
	box, edge := box2(t)
	faces := edgeFacesOf(t, box, edge)
	first := chamferSetbackOnFace(t, box, edge, faces[0], faces[0])
	second := chamferSetbackOnFace(t, box, edge, faces[1], faces[0])
	if stdmath.Abs(first-0.3) > 1e-6 {
		t.Errorf("with the reference on face 0 its setback = %g, want the FIRST distance 0.3", first)
	}
	if stdmath.Abs(second-0.6) > 1e-6 {
		t.Errorf("with the reference on face 1, face 0's setback = %g, want the SECOND distance 0.6", second)
	}
}

// chamferSetbackOnFace chamfers the edge 0.3/0.6 with the reference on referenceFace, and returns
// how far the cut set back along measureFace.
func chamferSetbackOnFace(t *testing.T, box *topo.Body, edge []byte, referenceFace, measureFace *topo.Face) float64 {
	t.Helper()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	ch := NewDressUpFeatures(fs).AddChamferDef(&ChamferDefinition{
		EdgeKeys: [][]byte{edge}, Distance: func() float64 { return 0.3 },
		Distance2: func() float64 { return 0.6 }, Type: types.ChamferTwoDistances, FlatCorners: true,
		ReferenceFace: referenceFace.ReferenceKey(),
	})
	fs.Recompute()
	if !ch.Health().OK() {
		t.Fatalf("referenced chamfer sick: %+v", ch.Health())
	}
	return setbackAlong(fs.Result()[0], measureFace)
}

// setbackAlong measures how far the chamfer cut back along a face, as the shortest distance from
// the original corner (2,2,0) to a new bottom vertex lying in that face's plane.
func setbackAlong(body *topo.Body, face *topo.Face) float64 {
	n := face.Geometry().NormalAt(0, 0)
	origin := centroidOf(faceVertexPoints(face))
	best := stdmath.Inf(1)
	for _, v := range body.Vertices() {
		p := v.Point()
		if stdmath.Abs(float64(p.Z)) > 1e-6 {
			continue
		}
		if off := origin.VectorTo(p).Dot(n); stdmath.Abs(float64(off)) > 1e-6 {
			continue // not on this face's plane
		}
		if d := float64(p.DistanceTo(math.P3(2, 2, 0))); d > 0.05 && d < best {
			best = d
		}
	}
	return best
}

// edgeFacesOf returns the two faces of the named edge.
func edgeFacesOf(t *testing.T, body *topo.Body, key []byte) []*topo.Face {
	t.Helper()
	e, ok := body.FindEdgeByKey(key)
	if !ok {
		t.Fatal("edge key did not resolve")
	}
	if f := e.Faces(); len(f) == 2 {
		return f
	}
	t.Fatal("edge does not bound exactly two faces")
	return nil
}

// TestPartialChamferBevelsOnlyItsSpan (#1888): a partial chamfer runs along part of the edge, so it
// removes proportionally less. The tool must also NOT overhang its interior ends, or it would bevel
// a little more edge than was asked for — the same rule a from-to hole's entry follows.
func TestPartialChamferBevelsOnlyItsSpan(t *testing.T) {
	t.Parallel()
	box, edge := box2(t)
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	ch := NewDressUpFeatures(fs).AddChamferDef(&ChamferDefinition{
		EdgeKeys: [][]byte{edge}, Distance: func() float64 { return 0.4 },
		PartialStart: func() float64 { return 0.5 }, PartialLength: func() float64 { return 1.0 },
	})
	fs.Recompute()
	if !ch.Health().OK() {
		t.Fatalf("partial chamfer sick: %+v", ch.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("partial chamfer not a valid solid: %+v", r)
	}
	want := 8 - 0.5*0.4*0.4*1.0 // the wedge over ONE cm of the 2 cm edge, not the whole of it
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; relErr(got, want) > 1e-6 {
		t.Errorf("partial chamfer volume = %g, want %g (the span, exactly — no overhang past it)", got, want)
	}
}
