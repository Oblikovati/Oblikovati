// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Hole placements (#1861). Each case drills into the same 4×4×2 block, so the volume removed says
// exactly how many bores landed and a coordinate assertion says where. The point of every placement
// is that it is a RULE re-resolved each recompute, not a frozen coordinate — the tests therefore
// check the resolved centre against the geometry the rule was written in terms of.

// holeBlock is a 4×4×2 block at the origin with its +Z top face returned alongside.
func holeBlock() (*topo.Body, []byte) {
	b := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}},
		sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	return b, b.Faces()[1].ReferenceKey() // the z=2 end cap, normal +Z
}

// drillWithPlacement runs a Ø2 × 1-deep blind hole under the given placement and returns the
// feature so the caller can read its health, plus the resulting bodies.
func drillWithPlacement(t *testing.T, p HolePlacement) (*PartFeature, []*topo.Body) {
	t.Helper()
	block, top := holeBlock()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 1 })
	hole.Definition().(*HoleFeature).Definition().Placement = p
	fs.Recompute()
	return hole, fs.Result()
}

// facetedBlindBoreVolume is what one Ø2 × 1-deep bore removes when the exact blind drill declines
// it and the faceted drillTool prism does the cut instead — the case for a bore that would clip a
// side face (the corner bores below stand 1 cm in from the block's edges, so a Ø2 bore is tangent
// to them). exactBlindBoreVolume is the same bore cut as a TRUE cylinder, πr²·depth; mass
// properties integrate that analytic face directly (M48/C3 #3453).
func facetedBlindBoreVolume() float64 { return drillToolPrismArea(1) * 1 }
func exactBlindBoreVolume() float64   { return stdmath.Pi * 1 * 1 * 1 }

// TestSketchPlacementDrillsOneHolePerCentrePoint is the placement that changes the shape of the
// feature: ONE hole feature, four bores. Before this a four-hole pattern had to be four features
// with four frozen coordinates.
func TestSketchPlacementDrillsOneHolePerCentrePoint(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3s): `make test-corpus`")
	}
	t.Parallel()
	sk := sketch.NewSketches().Add(topOfBlockPlane())
	for _, at := range [][2]float64{{1, 1}, {3, 1}, {3, 3}, {1, 3}} {
		sk.Points().Add(math.P2(math.Scalar(at[0]), math.Scalar(at[1]))).SetCenterPoint(true)
	}
	hole, res := drillWithPlacement(t, SketchHolePlacement{Sketch: sk})
	if !hole.Health().OK() {
		t.Fatalf("sketch-placed hole sick: %+v", hole.Health())
	}
	want := 32 - 4*facetedBlindBoreVolume()
	if got := query.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("volume = %g, want %g (block − FOUR Ø2×1 bores)", got, want)
	}
}

// TestSketchPlacementIgnoresPlainPoints is why the centre-point marker exists: a curve's endpoints
// live in the same collection, so drilling every point would bore a hole at each corner of any
// rectangle drawn in the sketch.
func TestSketchPlacementIgnoresPlainPoints(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(topOfBlockPlane())
	a, b := sk.Points().Add(math.P2(0.5, 0.5)), sk.Points().Add(math.P2(3.5, 0.5))
	sk.Lines().Add(a, b) // a and b are now a line's endpoints, not drill positions
	sk.Points().Add(math.P2(2, 2)).SetCenterPoint(true)

	hole, res := drillWithPlacement(t, SketchHolePlacement{Sketch: sk})
	if !hole.Health().OK() {
		t.Fatalf("sketch-placed hole sick: %+v", hole.Health())
	}
	want := 32 - exactBlindBoreVolume() // the single centre bore is clear of the sides: a true cylinder
	if got := query.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("volume = %g, want %g (ONE bore — the line's endpoints are not drill positions)", got, want)
	}
}

// TestSketchPlacementWithNoCentrePointsReportsWhy: a sketch with nothing marked would drill nothing
// at all, which reads as "the hole silently did not happen". It says what to fix instead.
func TestSketchPlacementWithNoCentrePointsReportsWhy(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(topOfBlockPlane())
	sk.Points().Add(math.P2(2, 2))
	hole, _ := drillWithPlacement(t, SketchHolePlacement{Sketch: sk})
	if hole.Health().OK() {
		t.Fatal("a sketch with no centre points drilled successfully; it must report the empty placement")
	}
	if reason := hole.Health().Reason; !strings.Contains(reason, "centre point") {
		t.Errorf("reason = %q, want it to name centre points", reason)
	}
}

// TestConcentricPlacementCentresOnACircularEdge: the bore takes the axis of the circle it is
// referenced to, so it stays concentric when that circle moves — the parametric anchor a frozen
// coordinate loses.
func TestConcentricPlacementCentresOnACircularEdge(t *testing.T) {
	t.Parallel()
	boss, capKey, rim, at := analyticBoss(t)
	sites, err := ConcentricHolePlacement{Face: HoleFaceRef{Key: capKey}, RefEdge: rim}.Sites(boss)
	if err != nil {
		t.Fatalf("concentric placement: %v", err)
	}
	if got := sites.Centers[0]; !nearPoint(got, at, 1e-9) {
		t.Errorf("concentric centre = %v, want %v — the referenced circle's axis, on the placement face", got, at)
	}
	if into := sites.Into.AsVector(); into.Dot(math.V3(0, 0, 1)) >= 0 {
		t.Errorf("drilling direction = %v, want it pointing INTO the +Z cap", into)
	}
}

// TestConcentricPlacementRefusesANonCircularEdge: a straight edge names no axis, so being
// "concentric" with it is meaningless. Falling back to, say, its midpoint would drill a hole that
// looks placed but tracks nothing.
func TestConcentricPlacementRefusesANonCircularEdge(t *testing.T) {
	t.Parallel()
	block, top := holeBlock()
	face, _ := block.FindFaceByKey(top)
	straight := edgeNearest(t, face, math.P3(2, 0, 2))
	_, err := ConcentricHolePlacement{Face: HoleFaceRef{Key: top}, RefEdge: straight}.Sites(block)
	if err == nil || !strings.Contains(err.Error(), "circle") {
		t.Errorf("a straight concentric reference gave %v; want an error saying only a circle names an axis", err)
	}
}

// analyticBoss is a Ø3 × 2 cylindrical boss standing off the origin — a true analytic solid, so it
// carries a real circular rim to be concentric with (a faceted prism's "rim" is line segments, and
// a placement anchored to one of those would track a facet rather than an axis). It returns the
// body, its top cap's key, the rim's key, and the rim's centre.
func analyticBoss(t *testing.T) (*topo.Body, []byte, []byte, math.Point3) {
	t.Helper()
	base := math.P3(1, 1, 0)
	body, err := brep.SolidCylinder(base, math.V3(0, 0, 1), 1.5, 2)
	if err != nil {
		t.Fatalf("boss: %v", err)
	}
	face, _ := planarCapAt(t, body, 2)
	for _, e := range face.Edges() {
		if c, ok := e.Geometry().(geom.Circle); ok {
			return body, face.ReferenceKey(), e.ReferenceKey(), c.Center
		}
	}
	t.Fatalf("the boss's top cap has no circular rim; got %d edges", len(face.Edges()))
	return nil, nil, nil, math.Point3{}
}

// planarCapAt finds the body's planar face whose centroid sits at height z, with that centroid.
func planarCapAt(t *testing.T, body *topo.Body, z float64) (*topo.Face, math.Point3) {
	t.Helper()
	for _, f := range body.Faces() {
		c := centroidOf(faceVertexPoints(f))
		if _, planar := f.Geometry().(geom.Plane); planar && stdmath.Abs(float64(c.Z)-z) < 1e-9 {
			return f, c
		}
	}
	t.Fatalf("no planar face at z=%g", z)
	return nil, math.Point3{}
}

// TestLinearPlacementLocatesByTwoOffsets: the dimensioned hole of a machining drawing. The subtle
// part is the SIDE the offsets run to — a distance from an edge is ambiguous until you say which
// way, and only one way lands on the part. Both corners of the block's top face are measured from
// to prove the offsets follow the face rather than a fixed axis sense.
func TestLinearPlacementLocatesByTwoOffsets(t *testing.T) {
	t.Parallel()
	block, top := holeBlock()
	face, ok := block.FindFaceByKey(top)
	if !ok {
		t.Fatal("top face key did not resolve")
	}
	for _, tc := range []struct {
		name         string
		from1, from2 math.Point3
		want         math.Point3
	}{
		{"near corner", math.P3(2, 0, 2), math.P3(0, 2, 2), math.P3(1, 1, 2)},
		{"far corner", math.P3(2, 4, 2), math.P3(4, 2, 2), math.P3(3, 3, 2)},
	} {
		p := LinearHolePlacement{
			Face:  HoleFaceRef{Key: top},
			Edge1: edgeNearest(t, face, tc.from1), Edge2: edgeNearest(t, face, tc.from2),
			Offset1: func() float64 { return 1 }, Offset2: func() float64 { return 1 },
		}
		sites, err := p.Sites(block)
		if err != nil {
			t.Fatalf("%s: linear placement: %v", tc.name, err)
		}
		if len(sites.Centers) != 1 {
			t.Fatalf("%s: linear placement gave %d sites, want 1", tc.name, len(sites.Centers))
		}
		if got := sites.Centers[0]; !nearPoint(got, tc.want, 1e-9) {
			t.Errorf("%s: linear centre = %v, want %v — one cm in from each reference edge", tc.name, got, tc.want)
		}
	}
}

// TestLinearPlacementRefusesParallelEdges: two parallel references name no point at all. Solving
// them anyway would divide by ~0 and drill somewhere far away.
func TestLinearPlacementRefusesParallelEdges(t *testing.T) {
	t.Parallel()
	block, top := holeBlock()
	face, _ := block.FindFaceByKey(top)
	along := edgeNearest(t, face, math.P3(2, 0, 2))
	p := LinearHolePlacement{
		Face: HoleFaceRef{Key: top}, Edge1: along, Edge2: along,
		Offset1: func() float64 { return 1 }, Offset2: func() float64 { return 2 },
	}
	if _, err := p.Sites(block); err == nil || !strings.Contains(err.Error(), "parallel") {
		t.Errorf("parallel reference edges gave %v; want an error saying they cross nowhere", err)
	}
}

// TestOnPointPlacementDrillsAlongItsAxis: the placement that needs no face, for a bore nothing on
// the body names. The direction comes from the work axis, and Flipped reverses it.
func TestOnPointPlacementDrillsAlongItsAxis(t *testing.T) {
	t.Parallel()
	wp := &WorkPoint{point: math.P3(2, 2, 2)}
	p := PointHolePlacement{Point: wp, Axis: yWorkAxis()}
	sites, err := p.Sites(nil)
	if err != nil {
		t.Fatalf("on-point placement: %v", err)
	}
	if !nearPoint(sites.Centers[0], math.P3(2, 2, 2), 1e-12) {
		t.Errorf("centre = %v, want the work point (2,2,2)", sites.Centers[0])
	}
	if got := sites.Into.AsVector(); !got.IsEqualTo(math.V3(0, 1, 0), 1e-12) {
		t.Errorf("direction = %v, want +Y (the axis it was told to drill along)", got)
	}
	flipped, err := PointHolePlacement{Point: wp, Axis: yWorkAxis(), Flipped: true}.Sites(nil)
	if err != nil {
		t.Fatalf("flipped on-point placement: %v", err)
	}
	if got := flipped.Into.AsVector(); !got.IsEqualTo(math.V3(0, -1, 0), 1e-12) {
		t.Errorf("flipped direction = %v, want −Y", got)
	}
}

// TestOnPointPlacementNeedsBoth: with no face to fall back on, a missing point or axis leaves the
// bore undefined rather than defaulted.
func TestOnPointPlacementNeedsBoth(t *testing.T) {
	t.Parallel()
	if _, err := (PointHolePlacement{Axis: yWorkAxis()}).Sites(nil); err == nil {
		t.Error("on-point placement with no work point resolved; it must report the missing point")
	}
	if _, err := (PointHolePlacement{Point: &WorkPoint{}}).Sites(nil); err == nil {
		t.Error("on-point placement with no axis resolved; it must report the missing axis")
	}
}

// topOfBlockPlane is the XY plane raised to the block's z=2 top face, where a placement sketch for
// these tests lives.
func topOfBlockPlane() sketch.Plane {
	xAxis, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	yAxis, _ := math.UnitVector3FromVector(math.V3(0, 1, 0))
	pl, _ := sketch.NewPlane(math.P3(0, 0, 2), xAxis, yAxis)
	return pl
}

// edgeNearest returns the reference key of the face edge whose midpoint sits closest to at — the
// unambiguous way to name one of a rectangle's four sides.
func edgeNearest(t *testing.T, face *topo.Face, at math.Point3) []byte {
	t.Helper()
	var best *topo.Edge
	bestD := math.Scalar(stdmath.Inf(1))
	for _, e := range face.Edges() {
		if d := e.Geometry().PointAt(0.5).VectorTo(at).Length(); d < bestD {
			best, bestD = e, d
		}
	}
	if best == nil {
		t.Fatalf("face has no edges to pick one near %v", at)
	}
	return best.ReferenceKey()
}

// nearPoint reports whether two points agree to tol.
func nearPoint(a, b math.Point3, tol float64) bool {
	return a.VectorTo(b).Length() <= math.Scalar(tol)
}

// TestHolePlacementRoundTrips is the point of persisting the RULE: a reopened part must still track
// its driving geometry. A recipe that saved the resolved coordinates instead would reopen with the
// holes frozen wherever they last landed — exactly the loss the placements exist to stop.
func TestHolePlacementRoundTrips(t *testing.T) {
	t.Parallel()
	sk := sketch.NewSketches().Add(topOfBlockPlane())
	sk.Points().Add(math.P2(1, 1)).SetCenterPoint(true)
	sk.Points().Add(math.P2(3, 3)).SetCenterPoint(true)
	// No base body: this case is about the RECIPE, and a base solid has no codec of its own.
	fs := NewPartFeatures(nil)
	hole := NewHoleFeatures(fs).AddDrilled([]byte("top"), func() float64 { return 1 }, func() float64 { return 1 })
	def := hole.Definition().(*HoleFeature).Definition()
	def.Placement = SketchHolePlacement{Sketch: sk, Flipped: true}
	def.Tap = HoleTapInfo{Tapped: true, Designation: "M6", Class: "6H", LeftHanded: true}
	def.Clearance = HoleClearanceInfo{Fastener: "M8", Fit: "free"}
	def.Type = SpotFaceHole

	data, err := fs.MarshalRecipe(oneSketch{s: sk})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	saved := data[0].Hole
	if saved.Placement == nil || saved.Placement.Kind != "sketch" || saved.Placement.Sketch != 1 {
		t.Fatalf("saved placement = %+v, want the sketch RULE (1-based index 1)", saved.Placement)
	}

	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{s: sk}, NewWorkGeometry()); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	back := fresh.Item(0).Definition().(*HoleFeature).Definition()
	p, ok := back.Placement.(SketchHolePlacement)
	if !ok || p.Sketch != sk || !p.Flipped {
		t.Errorf("restored placement = %#v, want the same sketch, still flipped", back.Placement)
	}
	if back.Type != SpotFaceHole {
		t.Errorf("restored seat = %v, want the spotface back (not collapsed into a counterbore)", back.Type)
	}
	if back.Tap != (HoleTapInfo{Tapped: true, Designation: "M6", Class: "6H", LeftHanded: true}) {
		t.Errorf("restored tap = %+v, want the whole tap alongside the seat", back.Tap)
	}
	if back.Clearance != (HoleClearanceInfo{Fastener: "M8", Fit: "free"}) {
		t.Errorf("restored clearance = %+v, want the FASTENER kept, not a resolved diameter", back.Clearance)
	}
}
