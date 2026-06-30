// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"
)

func TestCosmeticFeaturesPassBodiesThrough(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	c := NewCosmeticFeatures(fs)
	added := []Feature{
		c.AddDecal([]byte("face-1"), "logo.png"),
		c.AddReference("ref-A", []byte("src-1")),
		c.AddClient("com.acme.addin", map[string]string{"k": "v"}),
		c.AddMark([][]byte{[]byte("face-2")}, "LOT-42"),
		c.AddFinish([][]byte{[]byte("face-3")}, "Ra 1.6"),
	}
	fs.Recompute()

	// All five are healthy annotations and the single running solid is untouched.
	for _, f := range added {
		pf, ok := fs.ByID(patIDOf(fs, f))
		if !ok || !pf.Health().OK() {
			t.Errorf("%s feature not healthy", f.Kind())
		}
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("running bodies = %d, want 1 (cosmetic features add no geometry)", len(fs.Result()))
	}
}

func TestCosmeticFeaturesRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil)
	c := NewCosmeticFeatures(fs)
	c.AddDecal([]byte("face-1"), "logo.png")
	c.AddReference("ref-A", []byte("src-1"))
	c.AddClient("com.acme.addin", map[string]string{"slot": "3"})
	c.AddMark([][]byte{[]byte("face-2")}, "LOT-42")
	c.AddFinish([][]byte{[]byte("face-3")}, "Ra 1.6")

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 5 {
		t.Fatalf("feature count after round trip = %d, want 5", fresh.Count())
	}

	decal := fresh.Item(0).Definition().(*DecalFeature).Definition()
	if string(decal.FaceKey) != "face-1" || decal.Image != "logo.png" {
		t.Errorf("decal = %q %q, want face-1 logo.png", decal.FaceKey, decal.Image)
	}
	ref := fresh.Item(1).Definition().(*ReferenceFeature).Definition()
	if ref.Label != "ref-A" || string(ref.SourceKey) != "src-1" {
		t.Errorf("reference = %q %q, want ref-A src-1", ref.Label, ref.SourceKey)
	}
	client := fresh.Item(2).Definition().(*ClientFeature).Definition()
	if client.AddInID != "com.acme.addin" || client.Attributes["slot"] != "3" {
		t.Errorf("client = %q %v, want com.acme.addin slot=3", client.AddInID, client.Attributes)
	}
	mark := fresh.Item(3).Definition().(*MarkFeature).Definition()
	if len(mark.FaceKeys) != 1 || string(mark.FaceKeys[0]) != "face-2" || mark.Text != "LOT-42" {
		t.Errorf("mark = %v %q, want [face-2] LOT-42", mark.FaceKeys, mark.Text)
	}
	finish := fresh.Item(4).Definition().(*FinishFeature).Definition()
	if len(finish.FaceKeys) != 1 || string(finish.FaceKeys[0]) != "face-3" || finish.Spec != "Ra 1.6" {
		t.Errorf("finish = %v %q, want [face-3] Ra 1.6", finish.FaceKeys, finish.Spec)
	}
}
