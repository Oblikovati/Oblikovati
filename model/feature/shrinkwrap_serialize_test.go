// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// TestShrinkwrapSourceLinkAndOptionsRoundTrip round-trips a shrinkwrap through the feature
// codec — source identity link, linked flag, and every simplification option — restores it
// UNBOUND, rebinds a newer-revision source, and checks it flags out of date and
// re-simplifies the source into one enveloped body.
func TestShrinkwrapSourceLinkAndOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	block := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2)) // volume 8
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: math.Identity4(), Source: occFor("a:1")},
	}}
	link := DeriveSourceLink{Document: "asm.obk", InternalName: "GUID-1", DatabaseRevisionID: "rev-1"}
	def := ShrinkwrapDefinition{RemoveStyle: RemoveSmallParts, MinPartVolume: 0.5, EnvelopeStyle: EnvelopeWhole, PatchHoles: true}
	fs := NewPartFeatures(nil)
	pf := NewShrinkwrapComponents(fs).AddShrinkwrap(src, def, link)

	fd, err := serializeFeature(pf, nil, nil)
	if err != nil {
		t.Fatalf("serialize shrinkwrap: %v", err)
	}
	fs2 := NewPartFeatures(nil)
	restored, err := buildFeature(fs2, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("restore shrinkwrap: %v", err)
	}
	sw := restored.Definition().(*ShrinkwrapComponent)
	if sw.SourceLink() != link {
		t.Errorf("restored link = %+v, want %+v", sw.SourceLink(), link)
	}
	if sw.Options() != def {
		t.Errorf("restored options = %+v, want %+v", sw.Options(), def)
	}
	if sw.SourceVersion() != "" {
		t.Errorf("restored shrinkwrap should be unbound, got source version %q", sw.SourceVersion())
	}

	sw.BindSource(src, "rev-2")
	if !sw.OutOfDate() {
		t.Error("rebinding a newer-revision source should flag out of date")
	}
	fs2.Recompute()
	if len(fs2.Result()) != 1 {
		t.Fatalf("rebound shrinkwrap = %d bodies, want one enveloped body", len(fs2.Result()))
	}
}
