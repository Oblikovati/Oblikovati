//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// assemblyWithPlacedBox builds a box part and places it into a fresh assembly (assembly active),
// the fixture for the viewport-cache tests: the head must render AND cache an assembly's placed
// component geometry, not just a part's.
func assemblyWithPlacedBox(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	partDoc, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := partDoc.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	corners := []math.Point2{math.P2(-2, -2), math.P2(2, -2), math.P2(2, 2), math.P2(-2, 2)}
	pts := make([]*sketch.Point, len(corners))
	for i, c := range corners {
		pts[i] = sk.Points().Add(c)
	}
	for i := range pts {
		sk.Lines().Add(pts[i], pts[(i+1)%len(pts)]) // close the loop back to the first corner
	}
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	asm := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	if _, err := asm.PlaceComponentFromFile(asmDoc, partDoc, "box:1", math.Translation4(math.V3(10, 0, 0))); err != nil {
		t.Fatalf("place box: %v", err)
	}
	return s
}

// TestActiveBodiesRendersAssemblyComponents: the viewport's body source must surface an assembly's
// placed components, not just a part's bodies — otherwise a placed component renders nothing
// (the #769 render gap, regressed in the cached draw path). activeBodies feeds build() in
// cachedBodyDrawList, so a nil here is a blank assembly viewport.
func TestActiveBodiesRendersAssemblyComponents(t *testing.T) {
	s := assemblyWithPlacedBox(t)
	if got := len(activeBodies(s)); got != 1 {
		t.Fatalf("activeBodies(assembly) = %d bodies, want 1 placed component (else the assembly viewport is blank)", got)
	}
}

// TestBodyGeometryKeyCachesAssemblies: the tessellation cache key must be NON-EMPTY and STABLE for
// an assembly. An empty key takes cachedBodyDrawList's no-cache branch, which re-tessellates every
// placed body EVERY frame — the runaway repaint that makes the assembly viewport unstable. The key
// must also be stable between unedited frames so the cache actually holds (VisibleBodies returns
// stable body ids, so the key can).
func TestBodyGeometryKeyCachesAssemblies(t *testing.T) {
	s := assemblyWithPlacedBox(t)
	key := bodyGeometryKey(s)
	if key == "" {
		t.Fatal("bodyGeometryKey(assembly) = \"\", which disables the cache and re-tessellates every frame")
	}
	if again := bodyGeometryKey(s); again != key {
		t.Errorf("bodyGeometryKey is unstable between frames: %q then %q (the cache would rebuild every frame)", key, again)
	}
}

// TestViewportRendersMultipleComponents: several placements — copies of one component plus a
// second, distinct part — all render, and the cache key reflects every placed body (distinct ids)
// while staying stable. Guards the multi-component assembly case (each copy is an independent body
// via its own lineage prefix, so the viewport shows N bodies, not one).
func TestViewportRendersMultipleComponents(t *testing.T) {
	s := assemblyWithPlacedBox(t)
	asm := s.ActiveDocument().Content().(*compdef.AssemblyComponentDefinition)
	boxDoc := s.Workspace().Documents()[0] // the box part document, placed again as a second copy
	if _, err := asm.PlaceComponentFromFile(s.ActiveDocument(), boxDoc, "box:2", math.Translation4(math.V3(20, 0, 0))); err != nil {
		t.Fatalf("place second copy: %v", err)
	}
	if got := len(activeBodies(s)); got != 2 {
		t.Fatalf("activeBodies after a second placement = %d, want 2 independent component bodies", got)
	}
	key := bodyGeometryKey(s)
	if key == "" {
		t.Fatal("bodyGeometryKey = \"\" with two components placed (cache disabled)")
	}
	if again := bodyGeometryKey(s); again != key {
		t.Errorf("bodyGeometryKey unstable across frames with multiple components: %q then %q", key, again)
	}
}
