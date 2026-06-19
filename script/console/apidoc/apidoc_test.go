// SPDX-License-Identifier: GPL-2.0-only

package apidoc

import "testing"

func TestLookupAndSignature(t *testing.T) {
	s := New([]Doc{
		{Wire: "sketch.rectangle", Summary: "Draws a rectangle.", Params: []Param{
			{Name: "width", Type: "float64"}, {Name: "depth", Type: "float64"},
		}},
		{Wire: "documents.save", Summary: "Saves the document.", Params: nil},
	})

	d, ok := s.Lookup("sketch.rectangle")
	if !ok {
		t.Fatal("rectangle not found")
	}
	if d.Signature() != "sketch.rectangle{ width, depth }" {
		t.Errorf("signature = %q", d.Signature())
	}

	noArgs, _ := s.Lookup("documents.save")
	if noArgs.Signature() != "documents.save{}" {
		t.Errorf("no-arg signature = %q, want documents.save{}", noArgs.Signature())
	}

	if _, ok := s.Lookup("nope.missing"); ok {
		t.Error("unknown method should not be found")
	}
}

// TestDefaultIsPopulated guards that the build-time generated table is present and indexes a
// known method with its summary — so a missing/empty data_gen.go fails loudly.
func TestDefaultIsPopulated(t *testing.T) {
	s := Default()
	d, ok := s.Lookup("addins.activate")
	if !ok {
		t.Fatal("generated docs missing a known method (regenerate with go generate ./script/console/apidoc)")
	}
	if d.Summary == "" {
		t.Error("generated doc carries no summary")
	}
}
