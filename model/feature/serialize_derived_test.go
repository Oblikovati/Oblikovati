// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// sourceWithTwoBlocks builds a fake source of two disjoint unit blocks placed at a:1 (the
// origin) and a:2 (x=10), each carrying its occurrence path so derive styles can key on it.
func sourceWithTwoBlocks(t *testing.T) (*fakeAssemblySource, *occurrence.Occurrence, *occurrence.Occurrence) {
	t.Helper()
	block := solidBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	a, b := occFor("a:1"), occFor("a:2")
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: math.Identity4(), Source: a, Path: occurrence.OccurrencePath{"a:1"}},
		{Body: block, Transform: math.Translation4(math.V3(10, 0, 0)), Source: b, Path: occurrence.OccurrencePath{"a:2"}},
	}}
	return src, a, b
}

// roundTripDerive serializes pf's derived-assembly feature and rebuilds it into a fresh
// engine, returning the restored (still unbound) component and its engine.
func roundTripDerive(t *testing.T, pf *PartFeature) (*DerivedAssemblyComponent, *PartFeatures) {
	t.Helper()
	fd, err := serializeFeature(pf, nil, nil)
	if err != nil {
		t.Fatalf("serialize derive: %v", err)
	}
	fs := NewPartFeatures(nil)
	restored, err := buildFeature(fs, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("restore derive: %v", err)
	}
	return restored.Definition().(*DerivedAssemblyComponent), fs
}

// TestDerivedAssemblySourceLinkSurvivesRoundTrip checks the source identity link round-trips
// through the feature codec, restoring UNBOUND (no source) until a later BindSource.
func TestDerivedAssemblySourceLinkSurvivesRoundTrip(t *testing.T) {
	src, _, _ := sourceWithTwoBlocks(t)
	link := DeriveSourceLink{Document: "src.obk", InternalName: "GUID-1", DatabaseRevisionID: "rev-1"}
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, link)

	restored, _ := roundTripDerive(t, pf)
	if restored.SourceLink() != link {
		t.Errorf("restored link = %+v, want %+v", restored.SourceLink(), link)
	}
	if restored.SourceVersion() != "" {
		t.Errorf("restored derive should be unbound (empty source version), got %q", restored.SourceVersion())
	}
}

// TestDerivedAssemblyStyleSurvivesRoundTrip excludes one source occurrence, round-trips the
// derive, rebinds an equivalent source, and checks the excluded occurrence is still omitted
// — the base body is the single included block (volume 1, not 2).
func TestDerivedAssemblyStyleSurvivesRoundTrip(t *testing.T) {
	src, _, b := sourceWithTwoBlocks(t)
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{DatabaseRevisionID: "rev-1"})
	pf.Definition().(*DerivedAssemblyComponent).SetStyle(b, DeriveExclude)

	restored, fs2 := roundTripDerive(t, pf)
	// Rebind an equivalent source (same paths, fresh occurrence pointers) at the same revision.
	src2, _, _ := sourceWithTwoBlocks(t)
	restored.BindSource(src2, "rev-1")
	if restored.OutOfDate() {
		t.Error("same revision on rebind should not be out of date")
	}
	fs2.Recompute()
	if len(fs2.Result()) != 1 || !approx(volumeOf(fs2.Result()[0]), 1) {
		t.Fatalf("restored derive volume = %v bodies, want one body of volume 1 (a:2 excluded)", fs2.Result())
	}
}

// TestDeriveAcknowledgeSourceClearsOutOfDate checks AcknowledgeSource re-stamps the link's
// source revision and clears the out-of-date flag — the model side of deriveUpdate (#751).
func TestDeriveAcknowledgeSourceClearsOutOfDate(t *testing.T) {
	src, _, _ := sourceWithTwoBlocks(t)
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{DatabaseRevisionID: "rev-1"})
	d := pf.Definition().(*DerivedAssemblyComponent)
	d.BindSource(src, "rev-2")
	if !d.OutOfDate() {
		t.Fatal("binding a newer-revision source should flag out of date")
	}
	d.AcknowledgeSource("rev-2")
	if d.OutOfDate() {
		t.Error("AcknowledgeSource should clear out-of-date")
	}
	if got := d.SourceLink().DatabaseRevisionID; got != "rev-2" {
		t.Errorf("acknowledged link revision = %q, want rev-2", got)
	}
}

// TestDerivedAssemblyOutOfDateByRevision checks BindSource flags out-of-date exactly when
// the source's current revision differs from the one captured in the link.
func TestDerivedAssemblyOutOfDateByRevision(t *testing.T) {
	src, _, _ := sourceWithTwoBlocks(t)
	cases := []struct {
		name         string
		linkRev      string
		currentRev   string
		wantOutdated bool
	}{
		{"unchanged", "rev-1", "rev-1", false},
		{"edited", "rev-1", "rev-2", true},
		{"no link revision", "", "rev-2", false},
		{"no current revision", "rev-1", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fs := NewPartFeatures(nil)
			pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{DatabaseRevisionID: c.linkRev})
			d := pf.Definition().(*DerivedAssemblyComponent)
			d.BindSource(src, c.currentRev)
			if d.OutOfDate() != c.wantOutdated {
				t.Errorf("OutOfDate(link=%q,current=%q) = %v, want %v", c.linkRev, c.currentRev, d.OutOfDate(), c.wantOutdated)
			}
		})
	}
}
