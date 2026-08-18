// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// straightPathSketch is one open path along +X from the origin, the simplest rib profile: its
// in-plane LEFT normal is +Y, so side1 walls land at y>0 and side2 walls at y<0.
func straightPathSketch(length float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(length, 0))
	return s
}

// walledRib builds a rib over a straight 4-long path with thickness 1 and recomputes it, letting
// the caller set the wall options first. It returns the rib's own wall body.
func walledRib(t *testing.T, depth float64, tune func(*RibDefinition)) (*PartFeature, *topo.Body) {
	t.Helper()
	fs := NewPartFeatures(nil)
	def := &RibDefinition{
		Sketch: straightPathSketch(4), ProfileIndex: 0,
		Thickness: func() float64 { return 1 },
		Depth:     func() float64 { return depth },
		Operation: ops.NewBody,
	}
	tune(def)
	rib := &RibFeature{def: def}
	pf := fs.Add(rib)
	fs.Recompute()
	return pf, rib.tool
}

// validWall asserts the rib built a valid solid wall and returns it.
func validWall(t *testing.T, pf *PartFeature, wall *topo.Body, what string) *topo.Body {
	t.Helper()
	if !pf.Health().OK() {
		t.Fatalf("%s went sick: %+v", what, pf.Health())
	}
	if r := ops.Validate(wall); !r.Valid || !wall.IsSolid() {
		t.Fatalf("%s is not a valid solid: %+v", what, r)
	}
	return wall
}

// TestRibThickenSidePlacesWallOffThePath: side1/side2 move the whole wall to one side of the path
// instead of straddling it, without changing how much material it is (#1882).
func TestRibThickenSidePlacesWallOffThePath(t *testing.T) {
	for _, c := range []struct {
		name       string
		side       RibThickenSide
		minY, maxY float64
	}{
		{"symmetric", RibThickenSymmetric, -0.5, 0.5},
		{"side1", RibThickenSide1, 0, 1},
		{"side2", RibThickenSide2, -1, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			pf, wall := walledRib(t, 2, func(d *RibDefinition) { d.ThickenSide = c.side })
			body := validWall(t, pf, wall, "rib wall")
			b := body.RangeBox()
			if stdmath.Abs(float64(b.Min.Y)-c.minY) > 1e-9 || stdmath.Abs(float64(b.Max.Y)-c.maxY) > 1e-9 {
				t.Errorf("wall spans Y [%g, %g], want [%g, %g]", b.Min.Y, b.Max.Y, c.minY, c.maxY)
			}
			// 4 (length) × 1 (thickness) × 2 (depth), whichever side it sits on.
			if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-8) > 1e-6 {
				t.Errorf("wall volume = %g, want 8 — the side must not change the thickness", v)
			}
		})
	}
}

// draftTan is the test draft: tan 0.15, so over a depth of 2 the draft adds 0.3 to each side.
var draftTan = stdmath.Atan(0.15)

// TestRibDraftOpensTowardTheRoot: the draft must widen the wall at the ROOT — the end that lands
// on the part — for BOTH depth signs. A negative depth puts the root at the extrusion's base, so
// the taper the builder applies has to flip; without that flip the wall would open away from the
// part. The range box cannot see this (it reports the widest end either way), so the test probes
// the wall's actual half-width near each end (#1882).
func TestRibDraftOpensTowardTheRoot(t *testing.T) {
	for _, depth := range []float64{2, -2} {
		pf, wall := walledRib(t, depth, func(d *RibDefinition) { d.Draft = draftTan })
		body := validWall(t, pf, wall, "drafted rib wall")
		// Half-width is 0.5 at the sketch plane and 0.5+0.3 = 0.8 at the root, so y = 0.7 is
		// inside the wall near the root and outside it near the sketch plane.
		nearRoot := math.P3(2, 0.7, depth*0.95)
		nearSketch := math.P3(2, 0.7, depth*0.05)
		if !ops.PointInsideBody(body, nearRoot) {
			t.Errorf("depth %g: %v is outside the wall, but the draft should have widened the root", depth, nearRoot)
		}
		if ops.PointInsideBody(body, nearSketch) {
			t.Errorf("depth %g: %v is inside the wall, but the sketch-plane end holds the nominal thickness", depth, nearSketch)
		}
	}
}

// TestRibThicknessPlaneChoosesTheNominalEnd: holding the thickness at the root makes the ROOT
// exactly as thick as asked and thins the sketch-plane end, which is the whole observable
// difference between the two thickness planes (Inventor's RibThicknessPlaneEnum, #1882).
func TestRibThicknessPlaneChoosesTheNominalEnd(t *testing.T) {
	atSketch, wall := walledRib(t, 2, func(d *RibDefinition) { d.Draft = draftTan })
	sketchHeld := validWall(t, atSketch, wall, "thickness at the sketch plane").RangeBox()
	atRoot, rootWall := walledRib(t, 2, func(d *RibDefinition) {
		d.Draft, d.HoldThicknessAtRoot = draftTan, true
	})
	rootHeld := validWall(t, atRoot, rootWall, "thickness at the root").RangeBox()

	// Held at the sketch plane the wall grows past the nominal 1 (1 + 2×0.3); held at the root it
	// reaches exactly 1 there and is thinner everywhere else.
	if got := float64(sketchHeld.Max.Y - sketchHeld.Min.Y); stdmath.Abs(got-1.6) > 1e-9 {
		t.Errorf("thickness at the sketch plane: widest wall = %g, want 1.6 (1 + 2×0.3 of draft)", got)
	}
	if got := float64(rootHeld.Max.Y - rootHeld.Min.Y); stdmath.Abs(got-1) > 1e-9 {
		t.Errorf("thickness at the root: widest wall = %g, want exactly the nominal 1", got)
	}
	// The draft also lengthens the wall, i.e. the profile's ENDS are drafted too. Inventor makes
	// that a separate option (RibDefinition.DraftProfileEnds); we always draft them, and this
	// records it as measured behaviour rather than leaving it implied.
	if got := float64(sketchHeld.Max.X - sketchHeld.Min.X); stdmath.Abs(got-4.6) > 1e-9 {
		t.Errorf("drafted wall spans X %g, want 4.6 — the profile ends are drafted with the sides", got)
	}
}

// TestRibDraftRefusesAWallItWouldInvert: a draft deep enough to eat the whole thickness is a
// precise error, not a self-intersecting band (#1882).
func TestRibDraftRefusesAWallItWouldInvert(t *testing.T) {
	pf, _ := walledRib(t, 2, func(d *RibDefinition) {
		d.Draft, d.HoldThicknessAtRoot = stdmath.Atan(0.4), true // 2×2×0.4 = 1.6 > the thickness
	})
	if pf.Health().OK() {
		t.Error("a draft that consumes the whole thickness should go sick, not build an inverted wall")
	}
}

// TestRibExtendProfileLandsTheWallOnThePart: a path that stops short of the part is lengthened
// along its end tangent until it reaches it (Inventor's ExtendProfile, #1882). The box is
// extruded symmetrically so the sketch plane cuts through it — an in-plane ray must strike a
// side wall, not graze a cap.
func TestRibExtendProfileLandsTheWallOnThePart(t *testing.T) {
	for _, c := range []struct {
		name   string
		extend bool
		maxX   float64
	}{
		{"stops where it was drawn", false, 4},
		{"extends onto the part", true, 6},
	} {
		t.Run(c.name, func(t *testing.T) {
			fs := NewPartFeatures(nil)
			NewExtrudeFeatures(fs).AddExtrude(squareSketchAt(4, 6), []int{0}, ops.NewBody,
				Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 4 }}, 0)
			rib := &RibFeature{def: &RibDefinition{
				Sketch: straightPathSketch(4), ProfileIndex: 0,
				Thickness: func() float64 { return 1 },
				Depth:     func() float64 { return 2 },
				Operation: ops.NewBody, ExtendProfile: c.extend,
			}}
			pf := fs.Add(rib)
			fs.Recompute()
			wall := validWall(t, pf, rib.tool, "rib wall")
			if got := float64(wall.RangeBox().Max.X); stdmath.Abs(got-c.maxX) > 1e-6 {
				t.Errorf("wall reaches X %g, want %g (the part's near face is at X 6)", got, c.maxX)
			}
		})
	}
}

// TestRibWallOptionsRoundTrip: every wall option survives the recipe round-trip, and a document
// written before they existed reads back as the symmetric, undrafted wall it described.
func TestRibWallOptionsRoundTrip(t *testing.T) {
	sk := straightPathSketch(4)
	def := &RibDefinition{
		Sketch: sk, ProfileIndex: 0, Operation: ops.Join,
		Thickness: func() float64 { return 1 }, Depth: func() float64 { return 2 },
		ThickenSide: RibThickenSide2, Draft: draftTan,
		HoldThicknessAtRoot: true, ExtendProfile: true,
	}
	index := sketchList{sks: []*sketch.Sketch{sk}}
	data, err := serializeRib(def, index)
	if err != nil {
		t.Fatalf("serializeRib: %v", err)
	}
	if data.ThickenSide != "side2" || data.ThicknessAt != "root" || !data.ExtendProfile {
		t.Fatalf("persisted rib = %+v, want side2 / root / extendProfile", data)
	}
	restored, err := restoreRib(NewPartFeatures(nil), data, index)
	if err != nil {
		t.Fatalf("restoreRib: %v", err)
	}
	rdef := restored.Definition().(*RibFeature).Definition()
	if rdef.ThickenSide != RibThickenSide2 || !rdef.HoldThicknessAtRoot || !rdef.ExtendProfile ||
		stdmath.Abs(rdef.Draft-draftTan) > 1e-12 {
		t.Errorf("restored rib = %+v, want side2 / root / extend / draft %g", rdef, draftTan)
	}

	legacy := *data
	legacy.ThickenSide, legacy.ThicknessAt, legacy.Draft, legacy.ExtendProfile = "", "", 0, false
	old, err := restoreRib(NewPartFeatures(nil), &legacy, index)
	if err != nil {
		t.Fatalf("restoreRib (legacy): %v", err)
	}
	odef := old.Definition().(*RibFeature).Definition()
	if odef.ThickenSide != RibThickenSymmetric || odef.HoldThicknessAtRoot || odef.Draft != 0 {
		t.Errorf("legacy rib = %+v, want the symmetric undrafted wall", odef)
	}

	bad := *data
	bad.ThickenSide = "sideways"
	if _, err := restoreRib(NewPartFeatures(nil), &bad, index); err == nil {
		t.Error("an unknown thickenSide should be a precise error, not a silent fall back to symmetric")
	}
}
