// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// boxBodySession builds a session whose active part is a single extruded box, returning the
// session and the part definition so a test can read its keyed topology.
func boxBodySession(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "hl.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(pd); err != nil {
		t.Fatalf("activate: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0, c1 := sk.Points().Add(math.P2(0, 0)), sk.Points().Add(math.P2(4, 0))
	c2, c3 := sk.Points().Add(math.P2(4, 3)), sk.Points().Add(math.P2(0, 3))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	return s, def
}

// TestResolveRefOnBodiesBothForms covers the #157 reference resolution in the app package
// (it is otherwise only exercised across the package boundary by router tests, which Sonar's
// per-package coverage does not credit): a face/edge/vertex resolves from BOTH the raw
// reference-key form (model.referenceKeys) and the "face/…"/"vertex/…" form (model.selection).
func TestResolveRefOnBodiesBothForms(t *testing.T) {
	_, def := boxBodySession(t)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		t.Fatal("no bodies after recompute")
	}
	b := bodies[0]
	face, edge, vertex := b.Faces()[0], b.Edges()[0], b.Vertices()[0]

	cases := map[string]string{
		"raw face":     string(face.ReferenceKey()),
		"raw edge":     string(edge.ReferenceKey()),
		"raw vertex":   string(vertex.ReferenceKey()),
		"face/ form":   string(feature.FaceRef(face.ReferenceKey())),
		"vertex/ form": string(feature.VertexRef(vertex.ReferenceKey())),
	}
	for name, ref := range cases {
		if sel, ok := ResolveRefOnBodies(bodies, ref); !ok || sel == nil {
			t.Errorf("%s did not resolve (ref %q)", name, ref)
		}
	}
	if _, ok := ResolveRefOnBodies(bodies, "not-a-real-key"); ok {
		t.Error("a bogus reference resolved unexpectedly")
	}
}

// TestBodySelectionRefRoundTrips: a whole-body pick reports a non-empty body/<key> reference in
// Selection.References() (was "" — #1492), and that reference resolves back to the SAME body's
// BodyHandle through ResolveRefOnBodies, so an add-in can read and re-select a directly-picked body.
func TestBodySelectionRefRoundTrips(t *testing.T) {
	_, def := boxBodySession(t)
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		t.Fatal("no bodies after recompute")
	}
	b := bodies[0]

	sel := NewSelection()
	if !sel.Add(BodyHandle{Body: b}) {
		t.Fatal("adding a BodyHandle to the selection failed")
	}
	ref := sel.References()[0]
	if ref == "" {
		t.Fatal("a selected body still reports an empty reference (#1492 regression)")
	}
	if want := string(feature.BodyRef(b.ReferenceKey())); ref != want {
		t.Errorf("body ref = %q, want %q", ref, want)
	}

	resolved, ok := ResolveRefOnBodies(bodies, ref)
	if !ok {
		t.Fatalf("body ref %q did not resolve", ref)
	}
	h, isBody := resolved.(BodyHandle)
	if !isBody || h.Body != b {
		t.Errorf("resolved to %#v, want the original BodyHandle", resolved)
	}
}

// TestSessionResolveReference covers the session entry point, its no-part guard, and the lazily
// created highlight-set registry.
func TestSessionResolveReference(t *testing.T) {
	s, def := boxBodySession(t)
	if s.HighlightSets() == nil {
		t.Fatal("HighlightSets() returned nil")
	}
	face := def.SurfaceBodies().Item(0).Faces()[0]
	if _, ok := s.ResolveReference(string(feature.FaceRef(face.ReferenceKey()))); !ok {
		t.Error("session ResolveReference of a real face failed")
	}
	if _, ok := NewSession().ResolveReference("face/AAA"); ok {
		t.Error("ResolveReference with no active part should not resolve")
	}
}

// TestHighlightSetState covers the registry/accessors directly in the app package.
func TestHighlightSetState(t *testing.T) {
	hs := NewHighlightSets()
	set, err := hs.Create("guide", types.NewColor(255, 0, 255))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	set.AddItems("face/a", "face/a", "", "face/b") // dedupe + drop blank
	if set.Count() != 2 || len(set.Refs()) != 2 {
		t.Errorf("count = %d / refs %v, want 2 unique", set.Count(), set.Refs())
	}
	set.SetColor(types.NewColor(0, 255, 0))
	if set.Color().Hex() != "#00ff00" || set.Name() != "guide" {
		t.Errorf("set = %s %s, want guide #00ff00", set.Name(), set.Color().Hex())
	}
	if _, err := hs.Create("guide", types.NewColor(1, 2, 3)); err == nil {
		t.Error("duplicate name should fail")
	}
	if _, ok := hs.ByName("guide"); !ok || len(hs.All()) != 1 {
		t.Error("ByName/All wrong")
	}
	if !hs.Delete("guide") || hs.Delete("guide") {
		t.Error("Delete should succeed once")
	}
}
