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

// closedMobiusLoftBody builds a closed Möbius loft of n cross-sections (each twisted by half its
// azimuth) from the given section-sketch builder, asserts it is a valid solid, and returns the body
// — the shared spine of the Möbius design tests, so they don't each repeat the section/loft loop.
func closedMobiusLoftBody(t *testing.T, n int, radius, width, thick float64,
	section func(u, twist, radius, width, thick float64) *sketch.Sketch) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	sections := make([]LoftSection, n)
	for i := range n {
		u := 2 * stdmath.Pi * float64(i) / float64(n)
		sections[i] = LoftSection{Sketch: section(u, u/2, radius, width, thick), ProfileIndex: 0}
	}
	pf := NewLoftFeatures(fs).Add(sections, true, ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("Möbius loft went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("Möbius loft is not a valid solid: %+v", r)
	}
	return body
}

// centeredSquareOn returns a sketch on plane with a centered square of the given half
// width (corners ±half), wound counter-clockwise.
func centeredSquareOn(plane sketch.Plane, half float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	c0 := s.Points().Add(math.P2(-half, -half))
	c1 := s.Points().Add(math.P2(half, -half))
	c2 := s.Points().Add(math.P2(half, half))
	c3 := s.Points().Add(math.P2(-half, half))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

// planeAtZ returns the XY-parallel sketch plane at height z.
func planeAtZ(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), math.V3(1, 0, 0).AsUnit(), math.V3(0, 1, 0).AsUnit())
	return p
}

// centeredRectOn returns a sketch on plane with a centered halfW×halfH rectangle (corners
// ±halfW, ±halfH), wound counter-clockwise — an elongated section when halfW≠halfH.
func centeredRectOn(plane sketch.Plane, halfW, halfH float64) *sketch.Sketch {
	s := sketch.NewSketches().Add(plane)
	c0 := s.Points().Add(math.P2(-halfW, -halfH))
	c1 := s.Points().Add(math.P2(halfW, -halfH))
	c2 := s.Points().Add(math.P2(halfW, halfH))
	c3 := s.Points().Add(math.P2(-halfW, halfH))
	s.Lines().Add(c0, c1)
	s.Lines().Add(c1, c2)
	s.Lines().Add(c2, c3)
	s.Lines().Add(c3, c0)
	return s
}

// polygonAreaXY is the shoelace area of a loop projected to the XY plane.
func polygonAreaXY(loop []math.Point3) float64 {
	var sum float64
	n := len(loop)
	for i := range n {
		a, b := loop[i], loop[(i+1)%n]
		sum += float64(a.X*b.Y - b.X*a.Y)
	}
	if sum < 0 {
		sum = -sum
	}
	return sum / 2
}

// TestResampleLoopPreservesArea pins the loft volume-deficit fix (2026-06-15): resampling an
// elongated rectangle to a common point count must NOT cut its corners. A plain arc-length
// resample turned an 8×1 rectangle (area 8) into a 4.5-area quad (0.5625×); resampleLoop now
// preserves the corners, so the area is unchanged whether n equals or exceeds the vertex count.
func TestResampleLoopPreservesArea(t *testing.T) {
	t.Parallel()
	rect := []math.Point3{
		math.P3(-4, -0.5, 0), math.P3(4, -0.5, 0), math.P3(4, 0.5, 0), math.P3(-4, 0.5, 0),
	}
	const want = 8.0 // 8 wide × 1 tall
	for _, n := range []int{4, 8, 13, 32} {
		got := polygonAreaXY(resampleLoop(rect, n))
		if relErr(got, want) > 1e-9 {
			t.Errorf("resampleLoop(8×1 rect, n=%d) area = %g, want %g (corners must be preserved)", n, got, want)
		}
		if len(resampleLoop(rect, n)) != n {
			t.Errorf("resampleLoop(rect, n=%d) returned %d points, want %d", n, len(resampleLoop(rect, n)), n)
		}
	}
}

// mobiusSectionLoops builds n elongated-rectangle cross-sections around a ring of radius R, each
// twisted by `turns`·azimuth (turns=0.5 → a 0→180° half-twist = a Möbius band; turns=0 → a flat
// untwisted ring). W is the band width, T its thickness. CCW in the width/thickness frame.
func mobiusSectionLoops(n int, radius, width, thick, turns float64) [][]math.Point3 {
	loops := make([][]math.Point3, n)
	for i := range n {
		u := 2 * stdmath.Pi * float64(i) / float64(n)
		a := u * turns
		cu, su := stdmath.Cos(u), stdmath.Sin(u)
		ca, sa := stdmath.Cos(a), stdmath.Sin(a)
		w := math.V3(ca*cu, ca*su, sa)        // width direction
		td := math.V3(-sa*cu, -sa*su, ca)     // thickness direction
		c := math.P3(radius*cu, radius*su, 0) // section centre on the ring
		hw, ht := width/2, thick/2
		loops[i] = []math.Point3{
			c.TranslateBy(w.Scale(-hw)).TranslateBy(td.Scale(-ht)),
			c.TranslateBy(w.Scale(hw)).TranslateBy(td.Scale(-ht)),
			c.TranslateBy(w.Scale(hw)).TranslateBy(td.Scale(ht)),
			c.TranslateBy(w.Scale(-hw)).TranslateBy(td.Scale(ht)),
		}
	}
	return loops
}

// TestClosureShiftDetectsMonodromy pins the seam fix's core: a closed loft that twists 180° over a
// 180°-symmetric (rectangular) section comes back shifted by half its points; an untwisted ring
// does not. The closure (blend + mesh wrap) applies this offset so the seam doesn't pinch.
func TestClosureShiftDetectsMonodromy(t *testing.T) {
	t.Parallel()
	if got := closureShift(mobiusSectionLoops(12, 30, 16, 2, 0.5), true); got != 2 {
		t.Errorf("closureShift(Möbius rects) = %d, want 2 (rectangle 180° monodromy)", got)
	}
	if got := closureShift(mobiusSectionLoops(12, 30, 16, 2, 0), true); got != 0 {
		t.Errorf("closureShift(untwisted ring) = %d, want 0", got)
	}
	if got := closureShift(mobiusSectionLoops(12, 30, 16, 2, 0.5), false); got != 0 {
		t.Errorf("closureShift(open loft) = %d, want 0 (no wrap)", got)
	}
}

// TestClosedMobiusLoftClosesWithoutCram is the seam-notch regression: a closed loft twisting 180°
// around a ring must close as a clean watertight solid (volume ≈ W·T·2πR) AND must NOT cram the
// whole twist into the wrap segment — the old behaviour blew the wrap up to loftMaxSegmentSamples
// (a pinched notch); the monodromy-aware closure keeps every segment at the floor.
func TestClosedMobiusLoftClosesWithoutCram(t *testing.T) {
	t.Parallel()
	const n, R, W, T = 36, 30.0, 16.0, 2.0
	loops := mobiusSectionLoops(n, R, W, T, 0.5)

	secs := skinnedSections(loops, maxLoopCount(loops), true, loftEnds{}, loftGuides{})
	if len(secs) > n*(loftSegmentSamples+2) { // ~n·floor with the fix; a crammed wrap balloons this
		t.Errorf("closed twisted loft densified to %d sections — the seam is cramming (want ≈%d)", len(secs), n*loftSegmentSamples)
	}

	body, err := skinLoops(loops, true, "mobius", loftEnds{}, loftGuides{})
	if err != nil {
		t.Fatalf("skinLoops: %v", err)
	}
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("Möbius loft is not a valid solid: %+v", r)
	}
	wantV := W * T * 2 * stdmath.Pi * R // cross-section · centroid path length
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, wantV) > 0.03 {
		t.Errorf("Möbius volume = %g, want ≈%g (W·T·2πR)", v, wantV)
	}
}

// mobiusSectionSketch builds one Möbius cross-section as a sketch: a centered width×thick rectangle
// on a plane at azimuth u around a ring of radius `radius`, with its in-plane axes twisted by
// `twist` (width along cosθ·r̂+sinθ·ẑ, thickness perpendicular in the radial/axial plane) — the
// design's fixed-frame section. The plane's xAxis is the width direction, yAxis the thickness.
func mobiusSectionSketch(u, twist, radius, width, thick float64) *sketch.Sketch {
	cu, su := stdmath.Cos(u), stdmath.Sin(u)
	ca, sa := stdmath.Cos(twist), stdmath.Sin(twist)
	wdir := math.V3(ca*cu, ca*su, sa).AsUnit()   // width direction (plane xAxis)
	tdir := math.V3(-sa*cu, -sa*su, ca).AsUnit() // thickness direction (plane yAxis)
	center := math.P3(radius*cu, radius*su, 0)
	plane, _ := sketch.NewPlane(center, wdir, tdir)
	return centeredRectOn(plane, width/2, thick/2)
}

// TestLoftMobiusStripDesign is the kernel unit test for the loft built with this project's Möbius
// strip design parameters: 36 rectangular cross-sections (16×2 mm) on planes around a ring (R=30
// mm), each twisted by half the azimuth (a 180° half-twist over the loop), joined by a CLOSED loft.
// It drives the whole loft feature (profile → skin → solid) and pins the two 2026-06-15 fixes: the
// corner-preserving resample (full cross-section → right volume) and the monodromy-aware closure
// (seamless seam). A thin band of section w×t swept along the ring centroid (length 2πR) has
// volume w·t·2πR and one-sided surface area ≈ 2(w+t)·2πR, independent of the twist.
func TestLoftMobiusStripDesign(t *testing.T) {
	t.Parallel()
	const R, W, T = 3.0, 1.6, 0.2 // cm: ring 30 mm, band 16×2 mm (model units = cm)
	body := closedMobiusLoftBody(t, 36, R, W, T, mobiusSectionSketch)
	props := ops.BodyGeometryProperties(body, ops.DefaultQuality())
	if wantV := W * T * 2 * stdmath.Pi * R; relErr(props.Volume, wantV) > 0.03 { // 6.032 cm³
		t.Errorf("Möbius volume = %g cm³, want ≈%g (w·t·2πR); ~%g would mean corners are being cut",
			props.Volume, wantV, 0.5625*wantV)
	}
	if wantA := 2 * (W + T) * 2 * stdmath.Pi * R; relErr(props.Area, wantA) > 0.05 { // ≈67.86 cm²
		t.Errorf("Möbius area = %g cm², want ≈%g (2(w+t)·2πR)", props.Area, wantA)
	}
}

// mobiusSectionEllipseSketch is mobiusSectionSketch with an elliptical profile: an ellipse
// centered on the plane (semi-axes width/2 along the width direction, thick/2 across) — the
// rounded counterpart of the rectangular Möbius section.
func mobiusSectionEllipseSketch(u, twist, radius, width, thick float64) *sketch.Sketch {
	cu, su := stdmath.Cos(u), stdmath.Sin(u)
	ca, sa := stdmath.Cos(twist), stdmath.Sin(twist)
	wdir := math.V3(ca*cu, ca*su, sa).AsUnit()
	tdir := math.V3(-sa*cu, -sa*su, ca).AsUnit()
	center := math.P3(radius*cu, radius*su, 0)
	plane, _ := sketch.NewPlane(center, wdir, tdir)
	s := sketch.NewSketches().Add(plane)
	s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), math.Scalar(width/2), math.Scalar(thick/2))
	return s
}

// TestLoftMobiusStripEllipseDesign is the kernel unit test for the Möbius design with an ELLIPTICAL
// cross-section (the rectangle replaced by a 16×2 mm ellipse). A loft over a curved profile feeds
// the corner-preserving resample a finely-sampled loop and the closure the same 180° monodromy, so
// the rounded band must also close seamlessly with the right mass. An elliptical band of semi-axes
// a,b swept along the ring centroid has volume π·a·b·2πR.
func TestLoftMobiusStripEllipseDesign(t *testing.T) {
	t.Parallel()
	const R, W, T = 3.0, 1.6, 0.2 // cm: ring 30 mm, ellipse 16×2 mm
	body := closedMobiusLoftBody(t, 36, R, W, T, mobiusSectionEllipseSketch)
	a, b := W/2, T/2
	if wantV := stdmath.Pi * a * b * 2 * stdmath.Pi * R; relErr(ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume, wantV) > 0.05 {
		got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
		t.Errorf("elliptical Möbius volume = %g cm³, want ≈%g (π·a·b·2πR)", got, wantV)
	}
}

// TestLoftClosedTwistMeshStaysBounded guards the closed-twist over-tessellation fix: the seam
// monodromy must NOT be read as a per-segment twist. When it was, aroundSubdivisions saw the
// wrap as a ~180° twist and subdivided every cross-section ~12×, ballooning the ellipse Möbius to
// ~166k triangles and stalling the viewport (14 ms/frame just to flatten). The correct mesh tracks
// the loop density (≈ longitudinal sections × ellipse points); this pins it well under the blow-up.
func TestLoftClosedTwistMeshStaysBounded(t *testing.T) {
	t.Parallel()
	body := closedMobiusLoftBody(t, 36, 3.0, 1.6, 0.2, mobiusSectionEllipseSketch)
	mesh, _ := ops.TessellateBody(body, ops.DefaultQuality())
	if got := mesh.TriangleCount(); got > 30000 { // correct ≈14k; the monodromy bug produced ~166k
		t.Errorf("elliptical Möbius tessellated to %d triangles — the closed-twist seam is over-subdividing every section", got)
	}
}

func TestLoftElongatedRectKeepsVolume(t *testing.T) {
	t.Parallel()
	// An 8×1 rectangle lofted straight from z=0 to z=5 is a prism: V = area·h = 8·5 = 40.
	// The arc-length-resample bug skinned a 4.5-area quad → ~22.5; the corner-preserving
	// resample restores the full cross-section.
	fs := NewPartFeatures(nil)
	bottom := centeredRectOn(sketch.XYPlane(), 4, 0.5)
	top := centeredRectOn(planeAtZ(5), 4, 0.5)
	pf := NewLoftFeatures(fs).Add([]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}}, false, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("loft went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("lofted body is not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 40) > 0.02 {
		t.Errorf("elongated-rect prism volume = %g, want ≈40 (area 8 × height 5)", v)
	}
}

// sketchList is a SketchIndexer over an ordered set of sketches (loft uses several).
type sketchList struct{ sks []*sketch.Sketch }

func (l sketchList) IndexOf(s *sketch.Sketch) (int, bool) {
	for i, x := range l.sks {
		if x == s {
			return i, true
		}
	}
	return 0, false
}

func (l sketchList) At(i int) (*sketch.Sketch, bool) {
	if i < 0 || i >= len(l.sks) {
		return nil, false
	}
	return l.sks[i], true
}

func TestSweepAlongPathMakesValidSolid(t *testing.T) {
	t.Parallel()
	// A 2×2 square swept along an L-path (up Z, then over X) → a valid elbow solid.
	fs := NewPartFeatures(nil)
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(0, 0, 5)),
		sketch.NewPoint3D(math.P3(5, 0, 5)),
	}, false)
	pf := NewSweepFeatures(fs).Add(centeredSquareOn(sketch.XYPlane(), 1), 0, path, nil, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("sweep went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("swept body is not a valid solid: %+v", r)
	}
	// Cross-section area 4 along a path of length 10 → volume on the order of 40.
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v < 20 || v > 60 {
		t.Errorf("swept volume = %g, want roughly 40 (area 4 × path 10)", v)
	}
}

func TestLoftBetweenSquaresIsFrustum(t *testing.T) {
	t.Parallel()
	// A 4×4 square at z=0 lofted to a 2×2 square at z=5 → a square frustum:
	// V = h/3·(A1 + A2 + √(A1·A2)) = 5/3·(16 + 4 + 8) = 140/3 ≈ 46.667.
	fs := NewPartFeatures(nil)
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZ(5), 1)
	pf := NewLoftFeatures(fs).Add([]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}}, false, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("loft went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("lofted body is not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 140.0/3) > 0.02 {
		t.Errorf("frustum volume = %g, want ≈46.667", v)
	}
}

// planeAtZFlipped is planeAtZ with the normal reversed (xAxis (0,1,0), yAxis (1,0,0) → normal -Z),
// so a profile drawn on it is wound opposite a profile on an XY-parallel plane — the issue #1495
// condition where a user's two circles sat on oppositely-facing sketch planes.
func planeAtZFlipped(z float64) sketch.Plane {
	p, _ := sketch.NewPlane(math.P3(0, 0, z), math.V3(0, 1, 0).AsUnit(), math.V3(1, 0, 0).AsUnit())
	return p
}

// TestLoftBetweenOppositeNormalPlanesIsFrustum is the #1495 regression at the feature layer: a 4×4
// square at z=0 (plane normal +Z) lofted to a 2×2 square at z=5 whose sketch plane normal points -Z
// must still be the correct square frustum (V=140/3), not a winding-crossed bow-tie at ~1/3 the
// volume. matchWinding reverses the oppositely-wound top section so the ribs connect point-for-point.
func TestLoftBetweenOppositeNormalPlanesIsFrustum(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZFlipped(5), 1)
	pf := NewLoftFeatures(fs).Add([]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}}, false, ops.NewBody)
	fs.Recompute()

	if !pf.Health().OK() {
		t.Fatalf("opposite-normal loft went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("opposite-normal lofted body is not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErr(v, 140.0/3) > 0.02 {
		t.Errorf("opposite-normal frustum volume = %g, want ≈46.667 (a bow-tie would be ~1/3)", v)
	}
}

func TestSweepAndLoftRoundTrip(t *testing.T) {
	t.Parallel()
	prof := centeredSquareOn(sketch.XYPlane(), 1)
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZ(5), 1)
	idx := sketchList{sks: []*sketch.Sketch{prof, bottom, top}}

	fs := NewPartFeatures(nil)
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)), sketch.NewPoint3D(math.P3(0, 0, 5)),
	}, false)
	NewSweepFeatures(fs).Add(prof, 0, path, func() float64 { return 0.3 }, ops.Join)
	NewLoftFeatures(fs).Add([]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}}, false, ops.NewBody)

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	sweep := fresh.Item(0).Definition().(*SweepFeature).Definition()
	if sweep.Path().Count() != 2 || sweep.Twist() != 0.3 || sweep.Operation != ops.Join {
		t.Errorf("sweep round-trip lost data: pts=%d twist=%g op=%v", sweep.Path().Count(), sweep.Twist(), sweep.Operation)
	}
	loft := fresh.Item(1).Definition().(*LoftFeature).Definition()
	if len(loft.Sections) != 2 || loft.Sections[1].Sketch != top {
		t.Errorf("loft round-trip lost sections: %+v", loft.Sections)
	}
}

// TestLoftConditionsRoundTrip checks the S2 end conditions persist: a loft saved with an Angle
// start and a reversed-Direction end restores with the same conditions, angles, impacts and
// reversed flags (so a reopened .obk rebuilds the curved loft, not a ruled one).
func TestLoftConditionsRoundTrip(t *testing.T) {
	t.Parallel()
	bottom := centeredSquareOn(sketch.XYPlane(), 2)
	top := centeredSquareOn(planeAtZ(5), 1)
	idx := sketchList{sks: []*sketch.Sketch{bottom, top}}

	fs := NewPartFeatures(nil)
	first := LoftEnd{Condition: LoftAngle, Angle: 0.6, Impact: 1.5}
	last := LoftEnd{Condition: LoftDirection, Angle: 0.3, Impact: 2, Reversed: true}
	NewLoftFeatures(fs).addConditioned([]LoftSection{{Sketch: bottom, ProfileIndex: 0}, {Sketch: top, ProfileIndex: 0}}, false, ops.NewBody, first, last)

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	got := fresh.Item(0).Definition().(*LoftFeature).Definition()
	if got.First != first {
		t.Errorf("first condition round-trip = %+v, want %+v", got.First, first)
	}
	if got.Last != last {
		t.Errorf("last condition round-trip = %+v, want %+v", got.Last, last)
	}
}
