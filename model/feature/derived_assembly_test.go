// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// fakeAssemblySource is a local AssemblyBodySource (compdef.AssemblyComponentDefinition
// satisfies the real interface in production; importing compdef here would cycle).
type fakeAssemblySource struct {
	placed  []PlacedBody
	version int
}

func (f *fakeAssemblySource) PlacedBodies() []PlacedBody   { return f.placed }
func (f *fakeAssemblySource) ModelGeometryVersion() string { return fmt.Sprintf("v%d", f.version) }

// fakeOccDef is a minimal occurrence.Definition so a Source occurrence (for derive-style
// keying) can be minted; the derive transforms the PlacedBody, not this definition.
type fakeOccDef struct{}

func (fakeOccDef) RangeBox() math.Box { return math.EmptyBox() }

func occFor(name string) *occurrence.Occurrence {
	return occurrence.NewOccurrences().AddByComponentDefinition(name, fakeOccDef{}, math.Identity4())
}

func solidBlock(t *testing.T, min, max math.Point3) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(min, max, "src")
	if err != nil {
		t.Fatalf("SolidBlock(%v,%v): %v", min, max, err)
	}
	return b
}

func volumeOf(b *topo.Body) float64 {
	return ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
}

func approx(got, want float64) bool { d := got - want; return d < 1e-6 && d > -1e-6 }

// TestDerivedAssemblyMergesIncludedBodies is the F06 core: two placed source bodies are
// transformed into the part and merged into one base body (volume = sum).
func TestDerivedAssemblyMergesIncludedBodies(t *testing.T) {
	block := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2)) // volume 8
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: math.Identity4(), Source: occFor("a:1")},
		{Body: block, Transform: math.Translation4(math.V3(10, 0, 0)), Source: occFor("a:2")},
	}}
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{})
	fs.Recompute()
	if !pf.Health().OK() || len(fs.Result()) != 1 {
		t.Fatalf("derive: health=%+v bodies=%d, want ok and one base body", pf.Health(), len(fs.Result()))
	}
	if got := volumeOf(fs.Result()[0]); !approx(got, 16) {
		t.Errorf("derived volume = %g, want 16 (two 2³ blocks merged)", got)
	}
}

// TestDerivedAssemblySubtractsStyledOccurrence cuts a subtracted occurrence's body from
// the merged base — volume-gated against the analytic value.
func TestDerivedAssemblySubtractsStyledOccurrence(t *testing.T) {
	big := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))   // volume 8
	small := solidBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1)) // a 1³ corner of big
	cut := occFor("cut:1")
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: big, Transform: math.Identity4(), Source: occFor("keep:1")},
		{Body: small, Transform: math.Identity4(), Source: cut},
	}}
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{})
	pf.Definition().(*DerivedAssemblyComponent).SetStyle(cut, DeriveSubtract)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("derive health=%+v", pf.Health())
	}
	if got := volumeOf(fs.Result()[0]); !approx(got, 7) {
		t.Errorf("derived volume = %g, want 7 (8 minus a 1³ corner)", got)
	}
}

func TestDerivedAssemblyExcludesStyledOccurrence(t *testing.T) {
	kept := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	dropped := solidBlock(t, math.P3(10, 10, 10), math.P3(12, 12, 12))
	drop := occFor("drop:1")
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: kept, Transform: math.Identity4(), Source: occFor("keep:1")},
		{Body: dropped, Transform: math.Identity4(), Source: drop},
	}}
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{})
	pf.Definition().(*DerivedAssemblyComponent).SetStyle(drop, DeriveExclude)
	fs.Recompute()
	if got := volumeOf(fs.Result()[0]); !approx(got, 8) {
		t.Errorf("derived volume = %g, want 8 (excluded block omitted)", got)
	}
}

// TestDerivedAssemblyBreakLinkFreezesAndVersionTracks covers the associative pull (a
// source edit re-derives) and the break-link (the result freezes).
func TestDerivedAssemblyBreakLinkFreezesAndVersionTracks(t *testing.T) {
	block := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: math.Identity4(), Source: occFor("a:1")},
	}}
	fs := NewPartFeatures(nil)
	pf := NewDerivedAssemblyComponents(fs).AddDerived(src, DeriveSourceLink{})
	fs.Recompute()
	d := pf.Definition().(*DerivedAssemblyComponent)
	v0 := d.SourceVersion()

	// Edit the source → associative re-derive reflects the added body.
	src.placed = append(src.placed, PlacedBody{Body: block, Transform: math.Translation4(math.V3(10, 0, 0)), Source: occFor("a:2")})
	src.version++
	fs.MarkDirty(pf)
	fs.Recompute()
	if got := volumeOf(fs.Result()[0]); !approx(got, 16) {
		t.Fatalf("after source edit, volume = %g, want 16 (re-derived)", got)
	}
	if d.SourceVersion() == v0 {
		t.Error("source version did not advance on edit")
	}

	// Break the link and edit the source again → frozen, unchanged.
	if err := d.BreakLink(); err != nil {
		t.Fatalf("BreakLink: %v", err)
	}
	if d.Linked() {
		t.Error("Linked() is true after BreakLink")
	}
	src.placed = append(src.placed, PlacedBody{Body: block, Transform: math.Translation4(math.V3(20, 0, 0)), Source: occFor("a:3")})
	src.version++
	fs.MarkDirty(pf)
	fs.Recompute()
	if got := volumeOf(fs.Result()[0]); !approx(got, 16) {
		t.Errorf("after break-link, volume = %g, want frozen 16 (source change ignored)", got)
	}
}
