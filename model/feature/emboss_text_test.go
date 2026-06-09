// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// textOn builds a sketch on the given plane carrying one text box at (dx,dy).
func textOn(plane sketch.Plane, content string, height, dx, dy float64) (*sketch.Sketch, *sketch.TextBox) {
	s := sketch.NewSketches().Add(plane)
	tb := s.TextBoxes().Add(gmath.P2(gmath.Scalar(dx), gmath.Scalar(dy)), content, gmath.Scalar(height), 0, sketch.TextLeft)
	return s, tb
}

// TestTextEmbossRaisesMaterial proves an emboss that references a text entity recomputes to
// real raised geometry from the text's derived glyph profiles (not baked lines).
func TestTextEmbossRaisesMaterial(t *testing.T) {
	fs := embossedBlock(t)
	es, tb := textOn(planeAtZ(2), "I", 1.5, 2, 1) // a counter-less glyph keeps the test simple
	emb := NewEmbossFeatures(fs).AddText(es, tb, func() float64 { return 1 }, false, 0)
	fs.Recompute()
	if !emb.Health().OK() {
		t.Fatalf("text emboss went sick: %+v", emb.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("text-embossed body not valid: %+v", r)
	}
	// The boss adds material, so the volume must exceed the bare 10×10×2 block (200).
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v <= 200 {
		t.Errorf("text emboss volume = %g, want > 200 (raised glyph adds material)", v)
	}
}

// TestTextEmbossStoresReferenceNotGeometry is the BUG-FIX regression: an embossed text's
// recipe must store a REFERENCE to the text entity (sketch index + entity id) — never baked
// outline geometry — and round-trip back to a working text emboss.
func TestTextEmbossStoresReferenceNotGeometry(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	block := squareSketch(10)
	NewExtrudeFeatures(fs).AddByDistanceExtent(block, 0, ops.NewBody, func() float64 { return 2 })
	es, tb := textOn(planeAtZ(2), "AB", 2, 1, 1)
	NewEmbossFeatures(fs).AddText(es, tb, func() float64 { return 1 }, false, 0)

	data, err := fs.MarshalRecipe(twoSketches{a: block, b: es})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	emboss := lastEmboss(t, data)
	if emboss.TextEntity != uint64(tb.EntityID()) {
		t.Errorf("recipe textEntity = %d, want %d (the referenced text id)", emboss.TextEntity, tb.EntityID())
	}
	if len(emboss.Profiles) != 0 {
		t.Errorf("recipe carries %d profile indices, want 0 (text is by reference)", len(emboss.Profiles))
	}

	// The serialized bytes must NOT contain baked glyph coordinates — only the reference.
	out, err := yaml.Marshal(emboss)
	if err != nil {
		t.Fatalf("yaml: %v", err)
	}
	yml := string(out)
	for _, baked := range []string{"polygon", "outline", "coords", "points", "glyph"} {
		if strings.Contains(yml, baked) {
			t.Errorf("emboss recipe contains baked field %q:\n%s", baked, yml)
		}
	}

	// Round-trip: restore re-binds to the text entity and recomputes to a valid solid.
	fresh := NewPartFeatures(nil, nil)
	if err := fresh.ApplyRecipe(data, twoSketches{a: block, b: es}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	fresh.Recompute()
	if last := fresh.Item(fresh.Count() - 1); !last.Health().OK() || last.Kind() != "emboss" {
		t.Fatalf("restored emboss kind=%q health=%+v, want healthy emboss", last.Kind(), last.Health())
	}
}

// twoSketches is a SketchIndexer over an extrude sketch (0) and a text sketch (1).
type twoSketches struct{ a, b *sketch.Sketch }

func (o twoSketches) IndexOf(s *sketch.Sketch) (int, bool) {
	if s == o.a {
		return 0, true
	}
	if s == o.b {
		return 1, true
	}
	return 0, false
}
func (o twoSketches) At(i int) (*sketch.Sketch, bool) {
	switch i {
	case 0:
		return o.a, true
	case 1:
		return o.b, true
	}
	return nil, false
}

// lastEmboss returns the EmbossData of the last feature in a recipe.
func lastEmboss(t *testing.T, data []FeatureData) *EmbossData {
	t.Helper()
	last := data[len(data)-1]
	if last.Kind != "emboss" || last.Emboss == nil {
		t.Fatalf("last feature = %+v, want an emboss with payload", last)
	}
	return last.Emboss
}
