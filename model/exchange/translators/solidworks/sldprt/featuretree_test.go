// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "testing"

// node builds one feature-definition record: an optional class registration (its name abuts the
// object tag), then `<tag lo> <tag hi> ff fe ff <len> <name UTF-16LE>`. taghi carries the 0x80 tag
// bit; a value > 0x80 mimics a large part whose object index exceeds 255 (e.g. pistone's 0x81).
func node(class, name string, taghi byte) []byte {
	var b []byte
	if class != "" {
		b = append(b, 0xff, 0xff, 0x01, 0x00, byte(len(class)), 0x00) // MFC new-class header
		b = append(b, []byte(class)...)
	}
	b = append(b, 0x4b, taghi, 0xff, 0xfe, 0xff, byte(len([]rune(name))))
	for _, r := range name {
		b = append(b, byte(r), byte(r>>8))
	}
	b = append(b, 0x00, 0x00, 0x00, 0x00) // a little inter-node payload
	return b
}

// TestFeatureTreeDecode builds a synthetic feature-definition region mirroring the real grammar and
// checks the ordered, classified, audit-filtered result: registered classes are authoritative, a
// re-used class (empty registration) falls back to the localized name, dimension/origin nodes drop,
// and a large object-index tag (hi > 0x80) is still recognised.
func TestFeatureTreeDecode(t *testing.T) {
	var region []byte
	region = append(region, []byte(originFeatureClass)...) // region starts at the origin feature
	region = append(region, node("moOriginProfileFeature_c", "Origine", 0x80)...)
	region = append(region, node("moProfileFeature_c", "Schizzo1", 0x80)...)
	region = append(region, node("moExtrusion_c", "Estrusione1", 0x80)...)
	region = append(region, node("moLengthParameter_c", "D1", 0x80)...) // a dimension sub-node
	region = append(region, node("", "Estrusione2", 0x80)...)           // re-used moBoss class → by name
	region = append(region, node("moCut_c", "Taglio-Estrusione1", 0x80)...)
	region = append(region, node("Chamfer_c", "Smusso1", 0x81)...) // large object index

	// FeatureTree's body, run on the synthetic region (avoids constructing a Document/stream).
	var got []FeatureNode
	for _, n := range featureNameTags(region) {
		if k, keep := classifyFeature(n.class, n.name); keep {
			got = append(got, FeatureNode{Name: n.name, Kind: k})
		}
	}
	want := []FeatureNode{
		{"Schizzo1", KindSketch},
		{"Estrusione1", KindExtrude},
		{"Estrusione2", KindExtrude},
		{"Taglio-Estrusione1", KindCut},
		{"Smusso1", KindChamfer},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d features, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("feature %d = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestFeatureKindMaterial(t *testing.T) {
	material := []FeatureKind{KindExtrude, KindCut, KindRevolve, KindRevolveCut, KindMirror, KindCircularPattern, KindLinearPattern, KindHole}
	cosmetic := []FeatureKind{KindFillet, KindChamfer, KindDraft, KindSketch, KindUnknown}
	for _, k := range material {
		if !k.Material() {
			t.Errorf("%v: Material() = false, want true", k)
		}
	}
	for _, k := range cosmetic {
		if k.Material() {
			t.Errorf("%v: Material() = true, want false", k)
		}
	}
}

func TestClassifyFeature(t *testing.T) {
	cases := []struct {
		class, name string
		kind        FeatureKind
		keep        bool
	}{
		{"moExtrusion_c", "Estrusione1", KindExtrude, true},
		{"moBoss_c", "Estrusione2", KindExtrude, true},
		{"moCut_c", "Taglio-Estrusione1", KindCut, true},
		{"moRevolution_c", "Rivoluzione1", KindRevolve, true},
		{"Fillet_c", "Raccordo1", KindFillet, true},
		{"moDraft_c", "Sformo1", KindDraft, true},
		{"moMirrorPattern_c", "Specchia1", KindMirror, true},
		{"moProfileFeature_c", "Schizzo1", KindSketch, true},
		{"moOriginProfileFeature_c", "Origine", KindUnknown, false},
		{"moRefPlane_c", "Piano1", KindUnknown, false},
		{"moLengthParameter_c", "D1", KindUnknown, false},
		{"", "Taglio-Estrusione4", KindCut, true}, // re-used cut, by name (before extrude)
		{"", "Estrusione7", KindExtrude, true},    // re-used boss, by name
		{"", "Raccordo12", KindFillet, true},      // re-used fillet, by name
		{"", "D3", KindUnknown, false},            // dimension sub-node
		{"", "Corpi solidi", KindUnknown, false},  // an unrecognised folder-ish name
	}
	for _, c := range cases {
		k, keep := classifyFeature(c.class, c.name)
		if k != c.kind || keep != c.keep {
			t.Errorf("classifyFeature(%q,%q) = (%v,%v), want (%v,%v)", c.class, c.name, k, keep, c.kind, c.keep)
		}
	}
}
