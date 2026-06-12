// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The sweep definition union (M08 PBI-094, #314): taper, orientation modes,
// section twists, guide rail scaling, guide surface, solid sweep.

// straightZPath is a straight path up Z of the given length, sampled densely
// enough for the per-section machinery.
func straightZPath(length float64, samples int) *sketch.Path3D {
	pts := make([]*sketch.Point3D, samples+1)
	for i := 0; i <= samples; i++ {
		z := length * float64(i) / float64(samples)
		pts[i] = sketch.NewPoint3D(math.P3(0, 0, math.Scalar(z)))
	}
	return sketch.NewPath3D(pts, false)
}

func sweepDefRecompute(t *testing.T, fs *PartFeatures, def *SweepDefinition) *topo.Body {
	t.Helper()
	sf := &SweepFeature{def: def}
	pf := fs.Add(sf)
	pf.SetName(fs.UniqueName("Sweep"))
	sf.featName = pf.name
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("sweep sick: %+v", pf.Health())
	}
	body := fs.Result()[len(fs.Result())-1]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("swept body not a valid solid: %+v", r)
	}
	return body
}

// bodyVolume is the tessellated volume at display quality.
func bodyVolume(b *topo.Body) float64 {
	return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
}

// TestSweepTaperFrustum: a tapered straight sweep of a square is the analytic
// draft solid — side(s) = a + 2·tan(τ)·s, V = ∫ side² ds.
func TestSweepTaperFrustum(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	const a, L, taper = 2.0, 5.0, 0.1
	def := &SweepDefinition{
		Sketch: centeredSquareOn(sketch.XYPlane(), a/2), ProfileIndex: 0,
		Path:      func() *sketch.Path3D { return straightZPath(L, 16) },
		Taper:     func() float64 { return taper },
		Operation: ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	tt := stdmath.Tan(taper)
	want := a*a*L + 2*a*tt*L*L + 4.0/3*tt*tt*L*L*L // ∫(a+2τs)² ds
	if got := bodyVolume(body); relErr(got, want) > 0.01 {
		t.Errorf("tapered sweep volume = %g, want ≈%g", got, want)
	}
}

// TestSweepParallelOrientationShears: sweeping parallel-to-profile along a
// diagonal path SHEARS the prism (sections stay horizontal), so its volume is
// base area × HEIGHT, not base area × path length.
func TestSweepParallelOrientationShears(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	diag := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(3, 0, 4)), // length 5, height 4
	}, false)
	def := &SweepDefinition{
		Sketch: centeredSquareOn(sketch.XYPlane(), 1), ProfileIndex: 0,
		Path:        func() *sketch.Path3D { return diag },
		Orientation: types.ParallelToOriginalProfile,
		Operation:   ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	if got := bodyVolume(body); relErr(got, 4*4) > 0.02 { // area 4 × height 4
		t.Errorf("sheared parallel sweep volume = %g, want ≈16 (area×height)", got)
	}
}

// TestSweepSectionTwistsHold: a station table that twists to 90° by mid-path
// and holds keeps the prism volume (rigid sections) and stays valid.
func TestSweepSectionTwistsHold(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	def := &SweepDefinition{
		Sketch: centeredSquareOn(sketch.XYPlane(), 1), ProfileIndex: 0,
		Path: func() *sketch.Path3D { return straightZPath(6, 24) },
		TwistStations: []SweepTwistStation{
			{T: 0, Angle: 0}, {T: 0.5, Angle: stdmath.Pi / 2}, {T: 1, Angle: stdmath.Pi / 2},
		},
		Operation: ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	if got := bodyVolume(body); relErr(got, 4*6) > 0.03 {
		t.Errorf("section-twist sweep volume = %g, want ≈24", got)
	}
	if def.DefinitionType() != types.PathAndSectionTwistsSweepDef {
		t.Errorf("definition type = %v, want pathAndSectionTwists", def.DefinitionType())
	}
}

// TestSweepGuideRailScalesXY: a rail diverging linearly from the path scales
// the section linearly — the analytic square frustum.
func TestSweepGuideRailScalesXY(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	const a, L = 2.0, 6.0
	rail := func() *sketch.Path3D { // x = 2 at s=0 → x = 4 at s=L: scale 1 → 2
		return sketch.NewPath3D([]*sketch.Point3D{
			sketch.NewPoint3D(math.P3(2, 0, 0)),
			sketch.NewPoint3D(math.P3(4, 0, L)),
		}, false)
	}
	def := &SweepDefinition{
		Sketch: centeredSquareOn(sketch.XYPlane(), a/2), ProfileIndex: 0,
		Path:      func() *sketch.Path3D { return straightZPath(L, 16) },
		GuideRail: rail,
		Scaling:   types.XYProfileScaling,
		Operation: ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	// Square side a at the base, 2a at the top: the frustum V = L/3·(A1+A2+√(A1·A2)).
	a1, a2 := a*a, 4*a*a
	want := L / 3 * (a1 + a2 + stdmath.Sqrt(a1*a2))
	if got := bodyVolume(body); relErr(got, want) > 0.02 {
		t.Errorf("rail-scaled sweep volume = %g, want ≈%g (frustum)", got, want)
	}
	if def.DefinitionType() != types.PathAndGuideRailSweepDef {
		t.Errorf("definition type = %v, want pathAndGuideRail", def.DefinitionType())
	}
}

// TestSweepGuideRailNoScaling: the same diverging rail with scaling "none"
// keeps the prism size (the rail steers orientation only).
func TestSweepGuideRailNoScaling(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	rail := func() *sketch.Path3D {
		return sketch.NewPath3D([]*sketch.Point3D{
			sketch.NewPoint3D(math.P3(2, 0, 0)),
			sketch.NewPoint3D(math.P3(4, 0, 6)),
		}, false)
	}
	def := &SweepDefinition{
		Sketch: centeredSquareOn(sketch.XYPlane(), 1), ProfileIndex: 0,
		Path:      func() *sketch.Path3D { return straightZPath(6, 16) },
		GuideRail: rail,
		Scaling:   types.NoProfileScaling,
		Operation: ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	if got := bodyVolume(body); relErr(got, 4*6) > 0.02 {
		t.Errorf("unscaled rail sweep volume = %g, want ≈24", got)
	}
}

// TestSweepGuideRailTouchingPathSick: a rail crossing the path is a precise error.
func TestSweepGuideRailTouchingPathSick(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	rail := func() *sketch.Path3D {
		return sketch.NewPath3D([]*sketch.Point3D{
			sketch.NewPoint3D(math.P3(0, 0, 0)), // ON the path at s=0
			sketch.NewPoint3D(math.P3(2, 0, 6)),
		}, false)
	}
	def := &SweepDefinition{
		Sketch: centeredSquareOn(sketch.XYPlane(), 1), ProfileIndex: 0,
		Path:      func() *sketch.Path3D { return straightZPath(6, 8) },
		GuideRail: rail,
		Operation: ops.NewBody,
	}
	sf := &SweepFeature{def: def}
	pf := fs.Add(sf)
	fs.Recompute()
	if pf.Health().OK() {
		t.Fatal("a rail touching the path must go sick")
	}
}

// TestSweepGuideSurfaceConstantNormal: a planar guide face (constant normal)
// steers without distorting — prism volume preserved, type discriminated.
func TestSweepGuideSurfaceConstantNormal(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	// Base body whose +X face guides the sweep.
	base := centeredSquareOn(sketch.XYPlane(), 4)
	NewExtrudeFeatures(fs).AddByDistanceExtent(base, 0, ops.NewBody, func() float64 { return 1 })
	fs.Recompute()
	var guideKey []byte
	for _, f := range fs.Result()[0].Faces() {
		n := f.Geometry().NormalAt(0, 0)
		if float64(n.X) > 0.9 {
			guideKey = f.ReferenceKey()
		}
	}
	if guideKey == nil {
		t.Fatal("no +X face on the base body")
	}
	def := &SweepDefinition{
		Sketch: centeredSquareOn(planeAtZ(2), 1), ProfileIndex: 0,
		Path:         func() *sketch.Path3D { return straightZPathFrom(2, 6, 12) },
		GuideFaceKey: guideKey,
		Operation:    ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	if got := bodyVolume(body); relErr(got, 4*6) > 0.03 {
		t.Errorf("guide-surface sweep volume = %g, want ≈24", got)
	}
	if def.DefinitionType() != types.PathAndGuideSurfaceSweepDef {
		t.Errorf("definition type = %v, want pathAndGuideSurface", def.DefinitionType())
	}
}

// straightZPathFrom is a straight path from z0 up Z.
func straightZPathFrom(z0, length float64, samples int) *sketch.Path3D {
	pts := make([]*sketch.Point3D, samples+1)
	for i := 0; i <= samples; i++ {
		z := z0 + length*float64(i)/float64(samples)
		pts[i] = sketch.NewPoint3D(math.P3(0, 0, math.Scalar(z)))
	}
	return sketch.NewPath3D(pts, false)
}

// TestSolidSweepExtendsBox: dragging a unit-cube tool body along a straight
// 4-long X path sweeps the exact 5×1×1 envelope.
func TestSolidSweepExtendsBox(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(centeredSquareOn(sketch.XYPlane(), 0.5), 0, ops.NewBody, func() float64 { return 1 })
	fs.Recompute()
	toolIdx := 0
	xPath := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(4, 0, 0)),
	}, false)
	def := &SweepDefinition{
		Path:           func() *sketch.Path3D { return xPath },
		SolidToolIndex: &toolIdx,
		Operation:      ops.NewBody,
	}
	body := sweepDefRecompute(t, fs, def)
	if got := bodyVolume(body); relErr(got, 5*1*1) > 0.01 {
		t.Errorf("solid sweep envelope = %g, want 5 (1×1 cube dragged 4)", got)
	}
	if def.DefinitionType() != types.SolidSweepDef {
		t.Errorf("definition type = %v, want solid", def.DefinitionType())
	}
}
