// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// rectSketchOn returns a sketch on plane holding one closed rectangle over [x0,x1]×[y0,y1].
func rectSketchOn(plane sketch.Plane, x0, y0, x1, y1 float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	c0 := s.Points().Add(math.P2(math.Scalar(x0), math.Scalar(y0)))
	c1 := s.Points().Add(math.P2(math.Scalar(x1), math.Scalar(y0)))
	c2 := s.Points().Add(math.P2(math.Scalar(x1), math.Scalar(y1)))
	c3 := s.Points().Add(math.P2(math.Scalar(x0), math.Scalar(y1)))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

// TestEmbossFromPlaneRaisesOneSideAndCutsTheOther: the from-plane flavour is referenced to the
// SKETCH plane running through the part, not to a face, and does both booleans — it raises the
// region a depth on the plane's front side and cuts it the same depth on the back (#1893).
//
// The part is a block straddling the sketch plane (z ∈ [-2,2]); the profile overhangs it in X, so
// both halves are visible at once: past the block the raise is new material, and inside it the
// relief cut leaves a pocket.
func TestEmbossFromPlaneRaisesOneSideAndCutsTheOther(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddExtrude(squareSketch(4), []int{0}, ops.NewBody,
		Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 4 }}, 0)
	pf := NewEmbossFeatures(fs).Add(rectSketchOn(sketch.XYPlane(), 2, 1, 6, 3),
		[]int{0}, func() float64 { return 1 }, EmbossEngraveFromPlane, 0)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("from-plane emboss went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("from-plane emboss is not a valid solid: %+v", r)
	}
	// The raise carried the region past the block's X face at 4, out to the profile's 6.
	if got := float64(body.RangeBox().Max.X); stdmath.Abs(got-6) > 1e-6 {
		t.Errorf("part reaches X %g, want 6 — the raise should have carried the region past the block", got)
	}
	if p := math.P3(5, 2, 0.5); !ops.PointInsideBody(body, p) {
		t.Errorf("%v is empty, but the raise fills the region a depth in FRONT of the plane", p)
	}
	if p := math.P3(3, 2, -0.5); ops.PointInsideBody(body, p) {
		t.Errorf("%v is solid, but the relief cut empties the region a depth BEHIND the plane", p)
	}
	// Outside the profile the block is untouched on both sides of the plane.
	if p := math.P3(1, 2, -0.5); !ops.PointInsideBody(body, p) {
		t.Errorf("%v is empty, but it lies outside the profile and must be untouched", p)
	}
}

// TestEmbossFromPlaneRefusesOptionsItCannotHonour: the from-plane flavour works off the sketch
// plane, so there is no face to wrap onto, and it needs a part to raise and cut about.
func TestEmbossFromPlaneRefusesOptionsItCannotHonour(t *testing.T) {
	t.Run("with wrapToFace", func(t *testing.T) {
		fs := NewPartFeatures(nil)
		NewExtrudeFeatures(fs).AddExtrude(squareSketch(4), []int{0}, ops.NewBody,
			Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 4 }}, 0)
		pf := NewEmbossFeatures(fs).Add(rectSketchOn(sketch.XYPlane(), 1, 1, 3, 3),
			[]int{0}, func() float64 { return 1 }, EmbossEngraveFromPlane, 0)
		pf.Definition().(*EmbossFeature).Definition().WrapFaceKey = []byte("some-face")
		fs.Recompute()
		if pf.Health().OK() {
			t.Error("a from-plane emboss with wrapToFace should go sick: it has no face to wrap onto")
		}
	})
	t.Run("with no body", func(t *testing.T) {
		fs := NewPartFeatures(nil)
		pf := NewEmbossFeatures(fs).Add(rectSketchOn(sketch.XYPlane(), 1, 1, 3, 3),
			[]int{0}, func() float64 { return 1 }, EmbossEngraveFromPlane, 0)
		fs.Recompute()
		if pf.Health().OK() {
			t.Error("a from-plane emboss on an empty part should go sick: there is nothing to raise and cut about")
		}
	})
}

// shaftTangentPlane is the sketch plane tangent to a radius-r cylinder about +Z, touching it at
// angle 0 (the +X side) at height z. Its in-plane X runs AROUND the shaft and its Y runs ALONG it,
// so the plane normal points away from the axis — the setup a wrap needs.
func shaftTangentPlane(r, z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(math.Scalar(r), 0, math.Scalar(z)),
		math.V3(0, 1, 0).AsUnit(), math.V3(0, 0, 1).AsUnit())
	return p
}

// cylindricalFaceKey returns the reference key of the body's single cylindrical face.
func cylindricalFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, *geom.Cylinder:
			return f.ReferenceKey()
		}
	}
	t.Fatal("no cylindrical face on the body — the extruded circle did not stay analytic")
	return nil
}

// wrappedShaftEmboss raises a 3×2 sketch rectangle onto a radius-2 shaft and returns the pad tool.
func wrappedShaftEmboss(t *testing.T, depth float64, typ EmbossType) (*PartFeature, *EmbossFeature) {
	t.Helper()
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(circleOn(sketch.XYPlane(), 2), 0, ops.NewBody,
		func() float64 { return 4 })
	fs.Recompute()
	key := cylindricalFaceKey(t, fs.Result()[0])

	pf := NewEmbossFeatures(fs).Add(rectSketchOn(shaftTangentPlane(2, 2), -1.5, -1, 1.5, 1),
		[]int{0}, func() float64 { return depth }, typ, 0)
	pf.Definition().(*EmbossFeature).Definition().WrapFaceKey = key
	fs.Recompute()
	return pf, pf.Definition().(*EmbossFeature)
}

// radialSpanOf is the min and max distance of a body's vertices from the Z axis.
func radialSpanOf(b *topo.Body) (lo, hi float64) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range b.Vertices() {
		p := v.Point()
		r := stdmath.Hypot(float64(p.X), float64(p.Y))
		lo, hi = stdmath.Min(lo, r), stdmath.Max(hi, r)
	}
	return lo, hi
}

// TestWrappedEmbossHugsTheShaft: the point of the wrap is that the pad follows the curvature, so
// every one of its points sits between the shaft's surface and the surface plus the depth. A pad
// projected FLAT from the same tangent sketch fails this badly — its far corners stand at
// √(2² + 1.5²) = 2.5 from the axis and its base dives to 1.8 — which is what pins the assertion
// (#1893).
func TestWrappedEmbossHugsTheShaft(t *testing.T) {
	pf, emb := wrappedShaftEmboss(t, 0.2, EmbossFromFace)
	if !pf.Health().OK() {
		t.Fatalf("wrapped emboss went sick: %+v", pf.Health())
	}
	lo, hi := radialSpanOf(emb.tool)
	if lo < 2-0.02 || lo > 2+0.02 {
		t.Errorf("the pad's inner radius spans down to %g, want ≈2 — it must sit ON the shaft, not "+
			"chord through it", lo)
	}
	if stdmath.Abs(hi-2.2) > 0.02 {
		t.Errorf("the pad's outer radius reaches %g, want ≈2.2 (the shaft's 2 plus the 0.2 depth)", hi)
	}
	// Those bounds only see the loop's own points. Sampling BETWEEN them is what catches a pad whose
	// edges chord from vertex to vertex: mid-arc its surface would fall well inside 2.1.
	for _, arc := range []float64{-1.4, -0.7, 0, 0.7, 1.4} {
		a := arc / 2 // the angle that arc length subtends on the radius-2 shaft
		p := math.P3(math.Scalar(2.1*stdmath.Cos(a)), math.Scalar(2.1*stdmath.Sin(a)), 2)
		if !ops.PointInsideBody(emb.tool, p) {
			t.Errorf("arc %g: %v is outside the pad, so its surface does not follow the shaft there", arc, p)
		}
	}
}

// TestWrappedEmbossJoinsTheShaft: the wrapped pad has to end up part of the solid, standing the
// depth proud of the shaft all the way round the wrap.
func TestWrappedEmbossJoinsTheShaft(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(circleOn(sketch.XYPlane(), 2), 0, ops.NewBody,
		func() float64 { return 4 })
	fs.Recompute()
	key := cylindricalFaceKey(t, fs.Result()[0])
	pf := NewEmbossFeatures(fs).Add(rectSketchOn(shaftTangentPlane(2, 2), -1.5, -1, 1.5, 1),
		[]int{0}, func() float64 { return 0.2 }, EmbossFromFace, 0)
	pf.Definition().(*EmbossFeature).Definition().WrapFaceKey = key
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("wrapped emboss went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("shaft with a wrapped emboss is not a valid solid: %+v", r)
	}
	// Sample just outside the shaft at the wrap's near edge and at its far edge: a chorded pad
	// would have left the far edge short of the surface.
	for _, arc := range []float64{0, 1.4} { // arc length from the tangency, of the 1.5 available
		a := arc / 2 // the angle it subtends on the radius-2 shaft
		p := math.P3(math.Scalar(2.1*stdmath.Cos(a)), math.Scalar(2.1*stdmath.Sin(a)), 2)
		if !ops.PointInsideBody(body, p) {
			t.Errorf("arc %g: %v is empty, but the pad stands 0.2 proud of the shaft there", arc, p)
		}
	}
}

// TestWrapRefusesAFaceOrPlaneItCannotMap: the wrap is an isometry of a developable surface, so it
// is defined only on a cylinder reached from a TANGENT sketch plane. Everything else is refused
// out loud instead of being mapped as if it were a tangent cylinder, which would distort the
// profile silently (#1893).
func TestWrapRefusesAFaceOrPlaneItCannotMap(t *testing.T) {
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	t.Run("plane not parallel to the axis", func(t *testing.T) {
		if _, err := embossWrapFrameFor(cyl, sketch.XYPlane()); err == nil {
			t.Error("a sketch plane crossing the axis should be refused: its normal is the axis")
		}
	})
	t.Run("plane parallel but not tangent", func(t *testing.T) {
		if _, err := embossWrapFrameFor(cyl, shaftTangentPlane(3, 0)); err == nil {
			t.Error("a plane standing 3 from the axis of a radius-2 shaft should be refused")
		}
	})
	t.Run("tangent plane", func(t *testing.T) {
		fr, err := embossWrapFrameFor(cyl, shaftTangentPlane(2, 2))
		if err != nil {
			t.Fatalf("the tangent plane should be accepted: %v", err)
		}
		// The frame must anchor at the tangency and run around the shaft along +Y there.
		if fr.tangency != math.P2(0, 0) {
			t.Errorf("tangency in sketch coords = %v, want the origin", fr.tangency)
		}
		if got := float64(fr.circum.Dot(math.V3(0, 1, 0))); stdmath.Abs(got-1) > 1e-9 {
			t.Errorf("arc-length direction · +Y = %g, want 1 — the sketch's X runs around the shaft", got)
		}
	})
}

// TestWrappedEngraveRefusesCuttingPastTheAxis: an engrave deeper than the shaft's radius has no
// pad to build, so it is a precise error rather than a degenerate inside-out tool.
func TestWrappedEngraveRefusesCuttingPastTheAxis(t *testing.T) {
	pf, _ := wrappedShaftEmboss(t, 2.5, EngraveFromFace)
	if pf.Health().OK() {
		t.Error("a wrapped engrave deeper than the radius should go sick")
	}
}

// TestEmbossFlavourAndWrapRoundTrip: the flavour and the wrap face survive the recipe round-trip,
// and a document written before the flavours existed still reads as the raise or cut its `engrave`
// flag described.
func TestEmbossFlavourAndWrapRoundTrip(t *testing.T) {
	sk := squareSketch(4)
	index := sketchList{sks: []*sketch.Sketch{sk}}
	def := &EmbossDefinition{
		Sketch: sk, ProfileIndices: []int{0}, Depth: func() float64 { return 0.5 },
		Type: EmbossEngraveFromPlane, Taper: 0.2, WrapFaceKey: []byte("shaft-face"),
	}
	data, err := serializeEmboss(def, index)
	if err != nil {
		t.Fatalf("serializeEmboss: %v", err)
	}
	if data.Type != "fromPlane" || data.Engrave {
		t.Fatalf("persisted emboss = type %q engrave %v, want fromPlane / false", data.Type, data.Engrave)
	}
	restored, err := restoreEmboss(NewPartFeatures(nil), data, index)
	if err != nil {
		t.Fatalf("restoreEmboss: %v", err)
	}
	rdef := restored.Definition().(*EmbossFeature).Definition()
	if rdef.Type != EmbossEngraveFromPlane || rdef.Taper != 0.2 || string(rdef.WrapFaceKey) != "shaft-face" {
		t.Errorf("restored emboss = type %v taper %g wrap %q, want fromPlane / 0.2 / shaft-face",
			rdef.Type, rdef.Taper, rdef.WrapFaceKey)
	}

	// A raise and a cut must keep persisting through the legacy flag alone, so an older build still
	// reads a document this one wrote.
	for _, c := range []struct {
		typ     EmbossType
		engrave bool
	}{{EmbossFromFace, false}, {EngraveFromFace, true}} {
		def.Type = c.typ
		d, err := serializeEmboss(def, index)
		if err != nil {
			t.Fatalf("serializeEmboss(%v): %v", c.typ, err)
		}
		if d.Type != "" || d.Engrave != c.engrave {
			t.Errorf("%v persisted as type %q engrave %v, want no type key and engrave %v",
				c.typ, d.Type, d.Engrave, c.engrave)
		}
		back, err := restoreEmboss(NewPartFeatures(nil), d, index)
		if err != nil {
			t.Fatalf("restoreEmboss(%v): %v", c.typ, err)
		}
		if got := back.Definition().(*EmbossFeature).Definition().Type; got != c.typ {
			t.Errorf("%v read back as %v", c.typ, got)
		}
	}

	bad := *data
	bad.Type = "sideways"
	if _, err := restoreEmboss(NewPartFeatures(nil), &bad, index); err == nil {
		t.Error("an unknown emboss type should be a precise error, not a silent fall back to a raise")
	}
}

// TestWrappedEmbossHasCurvedCapFaces locks in the fix that the wrapped pad's inner/outer faces are
// genuine cylinder surfaces (not flat caps over a curved loop, which is an invalid non-planar-on-plane
// B-rep). The tool must be a valid solid carrying at least two cylindrical faces at the pad's radii.
func TestWrappedEmbossHasCurvedCapFaces(t *testing.T) {
	_, emb := wrappedShaftEmboss(t, 0.2, EmbossFromFace)
	if r := ops.Validate(emb.tool); !r.Valid || !emb.tool.IsSolid() {
		t.Fatalf("wrapped emboss tool must be a valid solid: %+v", r.Issues)
	}
	cyls := 0
	for _, f := range emb.tool.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyls++
		}
	}
	if cyls < 2 {
		t.Errorf("wrapped pad has %d cylindrical faces, want ≥2 (its inner and outer surfaces must be "+
			"true cylinders, not flat caps over a curved loop)", cyls)
	}
}
