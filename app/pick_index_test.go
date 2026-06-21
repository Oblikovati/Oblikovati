// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// boxRowAssembly places n copies of the [0,2]×[0,2]×[0,4] box part along X at 10-unit spacing,
// so each occupies a disjoint X span ([10k,10k+2]). A ray dropped down -Z through one box's
// footprint can only cross that one placement — the fixture for exercising the BVH's rejection
// of the placements a ray misses (M34-F5). The box part document is returned so a test can add
// a further placement and bump the occurrence revision.
func boxRowAssembly(t *testing.T, n int) (*Session, *compdef.AssemblyComponentDefinition, *doc.Document) {
	t.Helper()
	s, asm, boxDoc, asmDoc := emptyBoxAssembly(t)
	for k := 0; k < n; k++ {
		at := math.Translation4(math.V3(math.Scalar(10*k), 0, 0))
		if _, err := asm.PlaceComponentFromFile(asmDoc, boxDoc, "box", at); err != nil {
			t.Fatalf("place box %d: %v", k, err)
		}
	}
	return s, asm, boxDoc
}

// rayDownThroughBox returns a ray straight down -Z centered over box k's XY footprint.
func rayDownThroughBox(k int) (math.Point3, math.Vector3) {
	return math.P3(math.Scalar(10*k)+1, 1, 100), math.V3(0, 0, -1)
}

func TestPickIndexRayCrossesOnlyTheTargetedPlacement(t *testing.T) {
	_, asm, _ := boxRowAssembly(t, 6)
	idx := newAssemblyPickIndex(asm)
	origin, dir := rayDownThroughBox(3)
	bodies := idx.rayBodies(origin, dir)
	if len(bodies) != 1 {
		t.Fatalf("ray through one box crossed %d placements, want 1", len(bodies))
	}
	// The materialized body is the placement transformed into world space: box 3 sits at X=30.
	rb := bodies[0].RangeBox()
	if got := float64(rb.Min.X); got < 29.99 || got > 30.01 {
		t.Errorf("hit body min X = %g, want 30", got)
	}
	if _, ok := idx.occurrenceOf(bodies[0]); !ok {
		t.Error("hit body should resolve to its occurrence")
	}
}

func TestPickIndexRayMissReturnsNoCandidates(t *testing.T) {
	_, asm, _ := boxRowAssembly(t, 6)
	idx := newAssemblyPickIndex(asm)
	// Far to -X of every box, pointing down: crosses nothing.
	bodies := idx.rayBodies(math.P3(-50, 1, 100), math.V3(0, 0, -1))
	if len(bodies) != 0 {
		t.Errorf("ray missing all boxes crossed %d placements, want 0", len(bodies))
	}
}

func TestRayPickBodiesIsSubsetOfFullFlatten(t *testing.T) {
	s, asm, _ := boxRowAssembly(t, 8)
	full := s.worldAssemblyBodies(asm)
	origin, dir := rayDownThroughBox(2)
	picked := s.RayPickBodies(origin, dir)
	// The whole F5 win: a single pick materializes a handful, not all N.
	if len(picked) >= len(full) {
		t.Fatalf("RayPickBodies returned %d of %d bodies; should be a strict subset", len(picked), len(full))
	}
	if len(picked) == 0 {
		t.Fatal("ray through a box should pick at least one body")
	}
}

func TestRayPickBodiesNilForPart(t *testing.T) {
	// A part has no assembly, so there is no index and the picker must fall back to its body list.
	s := extrudedBox(t, 2, 4)
	if got := s.RayPickBodies(math.P3(0, 0, 10), math.V3(0, 0, -1)); got != nil {
		t.Errorf("RayPickBodies on a part = %v, want nil (fall back to full list)", got)
	}
}

func TestAssemblyPickIndexCachedByRevision(t *testing.T) {
	s, asm, boxDoc := boxRowAssembly(t, 4)
	first := s.assemblyPickIndexFor(asm)
	if again := s.assemblyPickIndexFor(asm); again != first {
		t.Error("index should be reused while the occurrence revision is unchanged")
	}
	// Adding a placement bumps the revision, so the index must rebuild.
	if _, err := asm.PlaceComponentFromFile(s.ActiveDocument(), boxDoc, "extra", math.Translation4(math.V3(99, 0, 0))); err != nil {
		t.Fatalf("place extra box: %v", err)
	}
	if rebuilt := s.assemblyPickIndexFor(asm); rebuilt == first {
		t.Error("index should rebuild after the occurrence revision changes")
	}
}

// TestPickThroughIndexResolvesOccurrence is the end-to-end check: a picker wired with the F5
// ray-body provider (as the head wires it) selects the same component a click would, proving the
// index path resolves a hit body to its occurrence exactly like the full-flatten path did (#769).
func TestPickThroughIndexResolvesOccurrence(t *testing.T) {
	s, asm, _ := boxRowAssembly(t, 6)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(50, 1, 2)    // looking down -X at box 3 ([30,32] on X)
	cam.Target = math.P3(31, 1, 2) // box 3's center column
	s.SetCamera(cam)
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return s.VisibleBodies() }).
		WithOccurrenceLookup(s.OccurrenceOfBody).
		WithRayBodies(s.RayPickBodies)

	sel, ok := p.Pick(200, 200, NewSelectionFilter()) // center pixel → ray at box 3
	if !ok {
		t.Fatal("Pick returned nothing aiming at a placed component")
	}
	occHandle, isOcc := sel.(OccurrenceHandle)
	if !isOcc {
		t.Fatalf("Pick returned %T, want OccurrenceHandle (assembly default selection)", sel)
	}
	found := false
	for _, occ := range asm.Occurrences().All() {
		if occ == occHandle.Occurrence {
			found = true
		}
	}
	if !found {
		t.Error("picked occurrence is not one of the assembly's placements")
	}
}

// countTransforms totals the instance transforms across groups — the number of draws the GPU
// would issue, the quantity frustum culling reduces.
func countTransforms(groups []InstanceGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Transforms)
	}
	return n
}

func TestCulledInstancesKeepsAllWhenFramed(t *testing.T) {
	s, _, _ := boxRowAssembly(t, 8) // boxes along X at 0,10,…,70
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(35, 1, 300) // far back, centered on the row → whole row on screen
	cam.Target = math.P3(35, 1, 2)
	if got, want := countTransforms(s.CulledInstances(cam)), countTransforms(s.VisibleInstances()); got != want {
		t.Errorf("framing the whole row culled to %d of %d transforms; should keep all", got, want)
	}
}

func TestCulledInstancesDropsOffscreenAndKeepsTarget(t *testing.T) {
	s, _, _ := boxRowAssembly(t, 8)
	full := countTransforms(s.VisibleInstances())
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 30) // zoomed onto box 0's column, looking down -Z
	cam.Target = math.P3(1, 1, 2)
	culled := s.CulledInstances(cam)
	if n := countTransforms(culled); n >= full || n == 0 {
		t.Fatalf("zoomed view kept %d of %d transforms; want a non-empty strict subset", n, full)
	}
	// Box 0 (translation X≈0) is dead center and must survive culling.
	keptBox0 := false
	for _, g := range culled {
		for _, tr := range g.Transforms {
			if x := float64(tr.Translation().X); x > -3 && x < 3 {
				keptBox0 = true
			}
		}
	}
	if !keptBox0 {
		t.Error("the box at the view center was culled (false negative — would pop)")
	}
}

// TestDetailCullingDropsSubPixelInstances pins F7: from very far every box projects sub-pixel and is
// dropped (only vertex throughput, no visible contribution); from close they are kept. VisibleInstances
// is never detail-culled, so bounds/framing still see the whole model.
func TestDetailCullingDropsSubPixelInstances(t *testing.T) {
	s, _, _ := boxRowAssembly(t, 8)
	full := countTransforms(s.VisibleInstances())
	if full == 0 {
		t.Fatal("fixture has no instances")
	}
	farCam := scene.NewCamera(400, 400)
	farCam.Eye = math.P3(35, 1, 200000) // every box is a fraction of a pixel from here
	farCam.Target = math.P3(35, 1, 2)
	if n := countTransforms(s.CulledInstances(farCam)); n != 0 {
		t.Errorf("from very far every box is sub-pixel; drew %d of %d, want 0", n, full)
	}
	nearCam := scene.NewCamera(400, 400)
	nearCam.Eye = math.P3(35, 1, 60) // boxes are tens of pixels here
	nearCam.Target = math.P3(35, 1, 2)
	if n := countTransforms(s.CulledInstances(nearCam)); n == 0 {
		t.Error("from close the boxes are well over a pixel and should be drawn")
	}
}

func TestCulledInstancesPartUnchanged(t *testing.T) {
	s := extrudedBox(t, 2, 4)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(10, 0, 2)
	cam.Target = math.P3(1, 1, 2)
	if got, want := countTransforms(s.CulledInstances(cam)), countTransforms(s.VisibleInstances()); got != want {
		t.Errorf("a part should not be culled: %d vs %d transforms", got, want)
	}
}

func TestWidestCentroidAxisAndCentroid(t *testing.T) {
	mk := func(boxes ...math.Box) *assemblyPickIndex {
		idx := &assemblyPickIndex{}
		for _, b := range boxes {
			idx.placements = append(idx.placements, pickPlacement{box: b})
		}
		idx.order = make([]int, len(boxes))
		for i := range idx.order {
			idx.order[i] = i
		}
		return idx
	}
	yIdx := mk(math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)), math.NewBox(math.P3(0, 9, 0), math.P3(1, 10, 1)))
	if got := yIdx.widestCentroidAxis(0, 2); got != 1 {
		t.Errorf("Y-dominant spread axis = %d, want 1", got)
	}
	zIdx := mk(math.NewBox(math.P3(0, 0, 0), math.P3(1, 1, 1)), math.NewBox(math.P3(0, 0, 9), math.P3(1, 1, 10)))
	if got := zIdx.widestCentroidAxis(0, 2); got != 2 {
		t.Errorf("Z-dominant spread axis = %d, want 2", got)
	}
	b := math.NewBox(math.P3(0, 2, 4), math.P3(2, 4, 8)) // center (1,3,6)
	if centroidOnAxis(b, 0) != 1 || centroidOnAxis(b, 1) != 3 || centroidOnAxis(b, 2) != 6 {
		t.Errorf("centroidOnAxis = (%g,%g,%g), want (1,3,6)",
			centroidOnAxis(b, 0), centroidOnAxis(b, 1), centroidOnAxis(b, 2))
	}
}

func TestCandidateBodiesFallsBackWhenIndexNil(t *testing.T) {
	full := []*topo.Body{{}, {}}
	p := &RayPicker{bodies: func() []*topo.Body { return full }}
	// No ray provider: use the full list.
	if got := p.candidateBodies(math.P3(0, 0, 0), math.V3(0, 0, 1)); len(got) != 2 {
		t.Fatalf("with no ray provider, candidateBodies = %d, want 2 (full list)", len(got))
	}
	// Provider returns nil (e.g. a part scene): still fall back to the full list.
	p.rayBodies = func(math.Point3, math.Vector3) []*topo.Body { return nil }
	if got := p.candidateBodies(math.P3(0, 0, 0), math.V3(0, 0, 1)); len(got) != 2 {
		t.Errorf("nil ray result should fall back to full list, got %d", len(got))
	}
	// Provider answers (even empty) — authoritative, do not fall back.
	p.rayBodies = func(math.Point3, math.Vector3) []*topo.Body { return []*topo.Body{} }
	if got := p.candidateBodies(math.P3(0, 0, 0), math.V3(0, 0, 1)); len(got) != 0 {
		t.Errorf("empty ray result is authoritative, got %d", len(got))
	}
}
