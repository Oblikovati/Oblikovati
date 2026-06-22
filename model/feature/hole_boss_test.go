// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

func TestHoleAndBossGenerateRealGeometry(t *testing.T) {
	// A drilled hole and a boss both generate real geometry (healthy). Each feature is
	// tested on its own body: a boolean rebuilds the topology with new lineage, so a
	// reference to a pre-cut face does not survive (chaining across a boolean is a follow-up).
	hb := prismBody()
	fsHole := NewPartFeatures(nil, nil)
	NewBaseFeatures(fsHole).AddBase(hb)
	drilled := NewHoleFeatures(fsHole).AddDrilled(hb.Faces()[0].ReferenceKey(), func() float64 { return 0.3 }, func() float64 { return 2 })
	fsHole.Recompute()
	if !drilled.Health().OK() {
		t.Errorf("hole health = %v, want OK (real cut)", drilled.Health())
	}

	bb := prismBody()
	fsBoss := NewPartFeatures(nil, nil)
	NewBaseFeatures(fsBoss).AddBase(bb)
	boss := NewBossFeatures(fsBoss).Add(bb.Faces()[0].ReferenceKey(), func() float64 { return 0.5 }, func() float64 { return 1 })
	fsBoss.Recompute()
	if !boss.Health().OK() {
		t.Errorf("boss health = %v, want OK (real stud, #327)", boss.Health())
	}

	tapped := NewHoleFeatures(NewPartFeatures(nil, nil)).AddTapped([]byte("f"), func() float64 { return 0.2 }, func() float64 { return 0.4 }, "M5x0.8")
	if tap := tapped.Definition().(*HoleFeature).Definition().Tap; !tap.Tapped || tap.Designation != "M5x0.8" {
		t.Errorf("tap info = %+v, want tapped M5x0.8", tap)
	}
}

// TestBossRaisesStudOfExactVolume: a boss on a block's top adds exactly the stud's prism
// volume (the entry overhang overlaps the block, so it adds nothing) — #327.
func TestBossRaisesStudOfExactVolume(t *testing.T) {
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=2 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	boss := NewBossFeatures(fs).Add(top, func() float64 { return 1 }, func() float64 { return 1.5 })
	fs.Recompute()
	if !boss.Health().OK() {
		t.Fatalf("boss sick: %+v", boss.Health())
	}
	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("boss result = %d bodies, want 1 (joined)", len(res))
	}
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("bossed body not a valid solid: %+v", r)
	}
	want := 32 + regularPolygonArea(0.5, holeFacets)*1.5
	if got := ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("bossed volume = %g, want %g (block + Ø1×1.5 stud)", got, want)
	}
}

// TestBossLostFaceSick: a boss whose placement face vanished goes Sick.
func TestBossLostFaceSick(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	boss := NewBossFeatures(fs).Add([]byte("gone"), func() float64 { return 1 }, func() float64 { return 1 })
	fs.Recompute()
	if boss.Health().Status != health.Sick {
		t.Errorf("boss health = %v, want Sick (lost placement face)", boss.Health().Status)
	}
}

// TestPatternOfBossReplicatesStuds: a rectangular pattern of a boss re-joins the clean stud
// (ToolBody) at each occurrence — one body whose volume grows by N−1 extra studs.
func TestPatternOfBossReplicatesStuds(t *testing.T) {
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey()

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	boss := NewBossFeatures(fs).Add(top, func() float64 { return 1 }, func() float64 { return 1.5 })
	// Step 1.2 keeps the copy fully on the block (a step to the rim would leave half the
	// entry overhang unsupported and add half a sliver of volume).
	NewPatternFeatures(fs).AddRectangular([]ID{boss.ID()},
		func() int { return 2 }, func() int { return 1 }, math.V3(1.2, 0, 0), noStep)
	fs.Recompute()
	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("pattern of a boss → %d bodies, want 1", len(res))
	}
	stud := regularPolygonArea(0.5, holeFacets) * 1.5
	want := 32 + 2*stud
	if got := ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("patterned-boss volume = %g, want %g (block + 2 studs)", got, want)
	}
}

func TestHoleDrillsThroughForReal(t *testing.T) {
	// A 4×4×2 block (vol 32). Drill a Ø2 hole through the top face → removes the
	// 32-gon cylinder of radius 1 over the full thickness 2.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // end cap (z=2), normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 3 }) // depth 3 > thickness 2 ⇒ through
	fs.Recompute()

	if !hole.Health().OK() {
		t.Fatalf("through hole sick: %+v", hole.Health())
	}
	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("hole result = %d bodies, want 1", len(res))
	}
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("drilled body not a valid solid: %+v", r)
	}
	want := 32 - regularPolygonArea(1, holeFacets)*2 // block − through cylinder
	if got := ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("drilled volume = %g, want %g (32 − Ø2 through hole)", got, want)
	}
}

func TestHoleDrillsAtExplicitCenter(t *testing.T) {
	// An 8×8×2 block. Drill a Ø2 through hole with an EXPLICIT off-centre drill point at
	// (2,3) instead of the face centroid (4,4): the bore must land there, not at the middle.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 8, Y: 0}, {X: 8, Y: 8}, {X: 0, Y: 8}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=2 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilledThrough(top, func() float64 { return 2 })
	center := math.P3(2, 3, 2)
	hole.feature.(*HoleFeature).def.Center = &center
	fs.Recompute()
	if !hole.Health().OK() {
		t.Fatalf("explicit-centre hole sick: %+v", hole.Health())
	}

	res := fs.Result()
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("drilled body not a valid solid: %+v", r)
	}
	var bore *geom.Cylinder
	for _, f := range res[0].Faces() {
		if c, ok := f.Geometry().(geom.Cylinder); ok {
			bore = &c
		}
	}
	if bore == nil {
		t.Fatal("no cylinder face: the through bore was not cut")
	}
	if stdmath.Abs(bore.Origin.X-2) > 1e-6 || stdmath.Abs(bore.Origin.Y-3) > 1e-6 {
		t.Errorf("bore axis at (%g,%g), want the explicit centre (2,3)", bore.Origin.X, bore.Origin.Y)
	}
	want := 64 - stdmath.Pi*1*1*2 // block − Ø2 through cylinder
	if got := ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume; (want-got)/want > 0.03 {
		t.Errorf("drilled volume = %g, want a hair under %g (64 − Ø2 through hole)", got, want)
	}
}

func TestHoleThroughAllProducesCylinderWall(t *testing.T) {
	// A 4×4×2 block, Ø2 hole through the top face. ThroughAll routes through the curved
	// boolean → a TRUE cylinder wall (one curved face), not a 32-gon prism.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=2 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilledThrough(top, func() float64 { return 2 })
	fs.Recompute()
	if !hole.Health().OK() {
		t.Fatalf("through-all hole sick: %+v", hole.Health())
	}
	res := fs.Result()
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("drilled body not a valid solid: %+v", r)
	}
	cylFaces := 0
	for _, f := range res[0].Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cylFaces++
		}
	}
	if cylFaces != 1 {
		t.Errorf("through-all hole has %d cylinder faces, want 1 (a true wall, not faceted)", cylFaces)
	}
	// Removed = an inscribed Ø2 cylinder over thickness 2: a hair under π·1²·2.
	bore := stdmath.Pi * 1 * 1 * 2
	removed := 32 - ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume
	if removed > bore+1e-9 || (bore-removed)/bore > 0.03 {
		t.Errorf("removed = %g, want a hair under %g (π·r²·h)", removed, bore)
	}
}

func TestBlindHoleProducesCylinderWallAndFlatBottom(t *testing.T) {
	// 4×4×2 block, Ø2 hole only 1 deep (blind) → exact cylinder wall + flat bottom disk.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=2 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 1 }) // depth 1 < 2 ⇒ blind
	fs.Recompute()
	if !hole.Health().OK() {
		t.Fatalf("blind hole sick: %+v", hole.Health())
	}
	res := fs.Result()
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("blind-drilled body not a valid solid: %+v", r)
	}
	cyl := 0
	for _, f := range res[0].Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyl++
		}
	}
	if cyl != 1 {
		t.Errorf("blind hole has %d cylinder faces, want 1 (true wall, not faceted)", cyl)
	}
	// Removed = an inscribed Ø2 cylinder 1 deep: a hair under π·1²·1.
	bore := stdmath.Pi * 1 * 1 * 1
	removed := 32 - ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume
	if removed > bore+1e-9 || (bore-removed)/bore > 0.03 {
		t.Errorf("removed = %g, want a hair under %g (π·r²·depth)", removed, bore)
	}
}

func TestCounterboreHoleProducesTwoWallsAndShoulder(t *testing.T) {
	// 8×8×4 block. Counterbore: Ø4 recess 1 deep + Ø2 bore through (total depth 4).
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 8, Y: 0}, {X: 8, Y: 8}, {X: 0, Y: 8}}, sketch.XYPlane(), span{near: 0, far: 4}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=4 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	cb := NewHoleFeatures(fs).AddCounterbore(top,
		func() float64 { return 2 }, func() float64 { return 4 }, // bore Ø2 × full depth
		func() float64 { return 4 }, func() float64 { return 1 }) // recess Ø4 × 1
	cb.Definition().(*HoleFeature).Definition().ThroughAll = true // bore goes through
	fs.Recompute()
	if !cb.Health().OK() {
		t.Fatalf("counterbore sick: %+v", cb.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("counterbored body not a valid solid: %+v", r)
	}
	cyl := 0
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			cyl++
		}
	}
	if cyl != 2 {
		t.Errorf("counterbore has %d cylinder faces, want 2 (recess wall + bore wall)", cyl)
	}
	// Removed = recess (Ø4 over the top 1) + bore (Ø2 over the remaining 3): inscribed, a hair
	// under the analytic sum (the bore only removes material below the shoulder).
	removed := 8.0*8.0*4.0 - ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	want := stdmath.Pi*2*2*1 + stdmath.Pi*1*1*3
	if removed > want+1e-9 || (want-removed)/want > 0.03 {
		t.Errorf("removed = %g, want a hair under %g (recess + bore)", removed, want)
	}
}

func TestCountersinkHoleProducesConeWall(t *testing.T) {
	// 10×10×6 block. Countersink: Ø4 sink at 90° included narrowing to a Ø2 bore through.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}, sketch.XYPlane(), span{near: 0, far: 6}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=6 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	cs := NewHoleFeatures(fs).AddCountersink(top,
		func() float64 { return 2 }, func() float64 { return 6 }, // bore Ø2 × full depth
		func() float64 { return 4 }, func() float64 { return stdmath.Pi / 2 }) // sink Ø4, 90°
	cs.Definition().(*HoleFeature).Definition().ThroughAll = true
	fs.Recompute()
	if !cs.Health().OK() {
		t.Fatalf("countersink sick: %+v", cs.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("countersunk body not a valid solid: %+v", r)
	}
	cone, cyl := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cone++
		case geom.Cylinder:
			cyl++
		}
	}
	if cone != 1 || cyl != 1 {
		t.Errorf("countersink has %d cone / %d cylinder faces, want 1 / 1 (sink wall + bore wall)", cone, cyl)
	}
}

func TestDrilledHoleWithConicalPoint(t *testing.T) {
	// 8×8×6 block, Ø2 blind hole 3 deep with a 118° drill point → cylinder bore + cone tip.
	block := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 8, Y: 0}, {X: 8, Y: 8}, {X: 0, Y: 8}}, sketch.XYPlane(), span{near: 0, far: 6}, 0, "blk")
	top := block.Faces()[1].ReferenceKey() // z=6 cap, normal +Z

	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(block)
	hole := NewHoleFeatures(fs).AddDrilled(top, func() float64 { return 2 }, func() float64 { return 3 })
	hole.Definition().(*HoleFeature).Definition().PointAngle = func() float64 { return 118 * stdmath.Pi / 180 }
	fs.Recompute()
	if !hole.Health().OK() {
		t.Fatalf("conical-point hole sick: %+v", hole.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("conical-point body not a valid solid: %+v", r)
	}
	cone, cyl := 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cone++
		case geom.Cylinder:
			cyl++
		}
	}
	if cone != 1 || cyl != 1 {
		t.Errorf("conical-point hole has %d cone / %d cylinder faces, want 1 / 1 (tip + bore)", cone, cyl)
	}
}

func TestHoleGoesSickOnLostFace(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	hole := NewHoleFeatures(fs).AddDrilled([]byte("ghost-face"), func() float64 { return 0.3 }, func() float64 { return 0.5 })
	fs.Recompute()
	if hole.Health().Status != health.Sick {
		t.Errorf("hole on a lost placement face = %v, want sick", hole.Health().Status)
	}
}

func TestHoleBossDefinitionAccessors(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	h := NewHoleFeatures(fs).AddDrilled([]byte("f"), func() float64 { return 1 }, func() float64 { return 2 })
	if h.Definition().(*HoleFeature).Definition().Type != DrilledHole {
		t.Error("hole definition not accessible")
	}
	b := NewBossFeatures(fs).Add([]byte("f"), func() float64 { return 1 }, func() float64 { return 2 })
	if b.Definition().(*BossFeature).Definition().Diameter() != 1 {
		t.Error("boss definition not accessible")
	}
}

// TestPatternOfHoleCutsEachOccurrence pins the fix that made a hole patternable: a hole exposes
// its drill cylinder as a [ToolFeature], so a pattern re-cuts the bore at every occurrence in
// one body — the earlier bug copied the whole solid into N bodies, and an unhandled source made
// the pattern a no-op. The same hole, alone vs. patterned 3×, must stay one solid yet remove ~2
// extra bores of material. (Bounds, not an exact volume: occurrence 0 is cut by the exact brep
// cutter while the replicated occurrences use the faceted drill tool, so the bores differ by a
// few percent — the band proves "two more holes" without over-fitting that gap.)
func TestPatternOfHoleCutsEachOccurrence(t *testing.T) {
	// A 16×4×2 block centred on X (spans −8..8): a Ø2 hole at the top-face centroid (x=0),
	// patterned 3× by +3 in X → bores at x=0,3,6, all clear of the x=±8 edges.
	corners := []math.Point2{{X: -8, Y: -2}, {X: 8, Y: -2}, {X: 8, Y: 2}, {X: -8, Y: 2}}
	blockVol := 16.0 * 4.0 * 2.0

	holeVol := func(t *testing.T, patterned bool) (float64, int) {
		t.Helper()
		fs := NewPartFeatures(nil, nil)
		block := buildPrism(corners, sketch.XYPlane(), span{near: 0, far: 2}, 0, "blk")
		NewBaseFeatures(fs).AddBase(block)
		hole := NewHoleFeatures(fs).AddDrilled(block.Faces()[1].ReferenceKey(), func() float64 { return 2 }, func() float64 { return 3 })
		if patterned {
			NewPatternFeatures(fs).AddRectangular([]ID{hole.ID()},
				func() int { return 3 }, func() int { return 1 }, math.V3(3, 0, 0), noStep)
		}
		fs.Recompute()
		res := fs.Result()
		if len(res) != 1 {
			t.Fatalf("patterned=%v → %d bodies, want 1", patterned, len(res))
		}
		if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
			t.Fatalf("patterned=%v body not a valid solid: %+v", patterned, r)
		}
		return ops.BodyGeometryProperties(res[0], ops.DefaultQuality()).Volume, len(res)
	}

	single, _ := holeVol(t, false)
	bore := blockVol - single // one bore's removed volume
	if bore <= 0 {
		t.Fatalf("single hole removed no material (vol %g >= block %g)", single, blockVol)
	}
	pattern, _ := holeVol(t, true)
	extra := single - pattern // the two replicated bores
	if extra < 1.5*bore || extra > 2.5*bore {
		t.Errorf("patterned hole removed %g extra (%.2f bores), want ~2 bores (one body, three holes)", extra, extra/bore)
	}
}
