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

// extentAlong is a body's extent (max − min of its vertices) projected onto a direction.
func extentAlong(b *topo.Body, dir math.Vector3) float64 {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range b.Vertices() {
		d := float64(v.Point().AsVector().Dot(dir))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
	}
	return hi - lo
}

func buildRib(t *testing.T, dir RibDirection) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0)) // path length 4 along +X
	rib := &RibFeature{def: &RibDefinition{
		Sketch: sk, ProfileIndex: 0,
		Thickness: func() float64 { return 1 }, // T
		Depth:     func() float64 { return 2 }, // d
		Direction: dir, Operation: ops.NewBody,
	}}
	pf := fs.Add(rib)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("rib (dir %d) went sick: %+v", dir, pf.Health())
	}
	return fs.Result()[0]
}

// TestRibParallelRotatesTheWall90 is the #2064 bug fix. A parallel (lateral) rib thickens the
// profile ALONG the plane normal and grows IN the plane — the opposite of the normal/web rib — so on
// the same path the two walls SWAP their thickness and depth between the plane normal (Z) and the
// in-plane perpendicular (Y). A parallel rib mistranslated as a normal one comes out rotated 90°.
func TestRibParallelRotatesTheWall90(t *testing.T) {
	t.Parallel()
	normal, parallel := buildRib(t, RibNormalToSketch), buildRib(t, RibParallelToSketch)
	for name, b := range map[string]*topo.Body{"normal": normal, "parallel": parallel} {
		if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
			t.Fatalf("%s rib is not a valid solid: %+v", name, r)
		}
		if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; stdmath.Abs(v-8) > 1e-6 {
			t.Errorf("%s rib volume = %g, want 8 (4×1×2 either way — only the orientation differs)", name, v)
		}
	}
	// The normal rib puts the DEPTH (2) along the plane normal Z and the THICKNESS (1) along the
	// in-plane perpendicular Y; the parallel rib swaps them.
	if nz, ny := extentAlong(normal, math.V3(0, 0, 1)), extentAlong(normal, math.V3(0, 1, 0)); stdmath.Abs(nz-2) > 1e-6 || stdmath.Abs(ny-1) > 1e-6 {
		t.Errorf("normal rib (Z=%g, Y=%g), want depth 2 along the normal and thickness 1 in-plane", nz, ny)
	}
	if pz, py := extentAlong(parallel, math.V3(0, 0, 1)), extentAlong(parallel, math.V3(0, 1, 0)); stdmath.Abs(pz-1) > 1e-6 || stdmath.Abs(py-2) > 1e-6 {
		t.Errorf("parallel rib (Z=%g, Y=%g), want thickness 1 ALONG the normal and depth 2 IN-plane (#2064)", pz, py)
	}
}

// TestRibParallelRefusesTaper mirrors Inventor: a taper (draft) is only allowed when the direction
// is normal to the sketch plane, so a parallel rib with a draft is refused rather than silently
// dropping it.
func TestRibParallelRefusesTaper(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	pf := fs.Add(&RibFeature{def: &RibDefinition{
		Sketch: sk, ProfileIndex: 0,
		Thickness: func() float64 { return 1 }, Depth: func() float64 { return 2 },
		Direction: RibParallelToSketch, Draft: 0.1, Operation: ops.NewBody,
	}})
	fs.Recompute()
	if pf.Health().OK() {
		t.Error("a parallel rib with a draft should go sick — a taper needs the normal direction")
	}
}

// TestRibDraftProfileEndsToggle: DraftProfileEnds defaults to drafted ends (nil), the square-ends
// case under a draft is refused honestly (not yet modelled), and without a draft the flag is moot.
func TestRibDraftProfileEndsToggle(t *testing.T) {
	t.Parallel()
	square := false
	drafted := true
	build := func(draftEnds *bool, draft float64) *PartFeature {
		fs := NewPartFeatures(nil)
		sk := sketch.NewSketches().Add(sketch.XYPlane())
		sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
		pf := fs.Add(&RibFeature{def: &RibDefinition{
			Sketch: sk, ProfileIndex: 0,
			Thickness: func() float64 { return 1 }, Depth: func() float64 { return 2 },
			Draft: draft, DraftProfileEnds: draftEnds, Operation: ops.NewBody,
		}})
		fs.Recompute()
		return pf
	}
	if pf := build(nil, 0.1); !pf.Health().OK() {
		t.Errorf("nil DraftProfileEnds (the default, drafted ends) with a draft went sick: %+v", pf.Health())
	}
	if pf := build(&drafted, 0.1); !pf.Health().OK() {
		t.Errorf("DraftProfileEnds=true with a draft went sick: %+v", pf.Health())
	}
	if pf := build(&square, 0.1); pf.Health().OK() {
		t.Error("DraftProfileEnds=false with a draft should be refused (square drafted ends are not modelled), not silently drafted")
	}
	if pf := build(&square, 0); !pf.Health().OK() {
		t.Errorf("DraftProfileEnds=false WITHOUT a draft is a square-ended wall either way and must build: %+v", pf.Health())
	}
}

// TestRibParallelToNextGrowsOntoThePart exercises the parallel to-next path: the wall grows in the
// plane, perpendicular to the path, toward the side the material is on, until it lands. A profile
// above a box grows DOWN onto the box top.
func TestRibParallelToNextGrowsOntoThePart(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(squareSketch(6), 0, ops.NewBody, func() float64 { return 4 })
	fs.Recompute()
	boxVol := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume

	// A section plane through the box (y = 3), X in-plane, Z in-plane; a path above the box top.
	plane, err := sketch.NewPlane(math.P3(0, 3, 0), math.V3(1, 0, 0).AsUnit(), math.V3(0, 0, 1).AsUnit())
	if err != nil {
		t.Fatal(err)
	}
	sk := sketch.NewSketches().Add(plane)
	sk.Lines().AddByTwoPoints(math.P2(1, 5), math.P2(5, 5)) // x∈[1,5], z=5 — above the box top (z=4)
	pf := fs.Add(&RibFeature{def: &RibDefinition{
		Sketch: sk, ProfileIndex: 0,
		Thickness: func() float64 { return 1 },
		ToNext:    true, Direction: RibParallelToSketch, Operation: ops.Join,
	}})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("parallel to-next rib went sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("parallel to-next rib is not a valid solid: %+v", r.Issues)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= boxVol {
		t.Errorf("parallel to-next rib added no material (volume %g ≤ box %g) — it did not grow onto the part", v, boxVol)
	}
	// The wall grew from the profile (z=5) down onto the box top (z=4), so the body still reaches z=5.
	if hi := float64(body.RangeBox().Max.Z); stdmath.Abs(hi-5) > 1e-6 {
		t.Errorf("body reaches Z %g, want 5 — the rib should stand from the profile down to the box top", hi)
	}
}

// TestRibDirectionRoundTrip round-trips the #2064 fields: a parallel rib with square ends persists
// and restores, a legacy recipe (no direction) reads back as the normal/web default, and an unknown
// direction is a precise error rather than a silent 90° rotation.
func TestRibDirectionRoundTrip(t *testing.T) {
	t.Parallel()
	sk := straightPathSketch(4)
	square := false
	def := &RibDefinition{
		Sketch: sk, ProfileIndex: 0, Operation: ops.Join,
		Thickness: func() float64 { return 1 }, Depth: func() float64 { return 2 },
		Direction: RibParallelToSketch, DraftProfileEnds: &square,
	}
	index := sketchList{sks: []*sketch.Sketch{sk}}
	data, err := serializeRib(def, index)
	if err != nil {
		t.Fatalf("serializeRib: %v", err)
	}
	if data.Direction != "parallel" || data.DraftProfileEnds == nil || *data.DraftProfileEnds {
		t.Fatalf("persisted rib = %+v, want direction parallel + draftProfileEnds false", data)
	}
	restored, err := restoreRib(NewPartFeatures(nil), data, index)
	if err != nil {
		t.Fatalf("restoreRib: %v", err)
	}
	rdef := restored.Definition().(*RibFeature).Definition()
	if rdef.Direction != RibParallelToSketch || rdef.DraftProfileEnds == nil || *rdef.DraftProfileEnds {
		t.Errorf("restored rib = %+v, want parallel + square ends", rdef)
	}

	legacy := *data
	legacy.Direction, legacy.DraftProfileEnds = "", nil
	old, err := restoreRib(NewPartFeatures(nil), &legacy, index)
	if err != nil {
		t.Fatalf("restoreRib (legacy): %v", err)
	}
	if odef := old.Definition().(*RibFeature).Definition(); odef.Direction != RibNormalToSketch || odef.DraftProfileEnds != nil {
		t.Errorf("legacy rib = %+v, want the normal/web rib with the default (nil) draftProfileEnds", odef)
	}

	bad := *data
	bad.Direction = "sideways"
	if _, err := restoreRib(NewPartFeatures(nil), &bad, index); err == nil {
		t.Error("an unknown direction should be a precise error, not a silent fall back to normal")
	}
}
