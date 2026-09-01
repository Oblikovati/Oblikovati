// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// cavityBlock builds a 4³ block with a fully enclosed 2³ cavity (volume 56) by cutting
// an inner block from a bigger one — a hollow part for the hole-patch gate.
func cavityBlock(t *testing.T) *topo.Body {
	t.Helper()
	big := solidBlock(t, math.P3(0, 0, 0), math.P3(4, 4, 4))
	small := solidBlock(t, math.P3(1, 1, 1), math.P3(3, 3, 3))
	res, err := ops.Boolean(ops.Cut, big, small)
	if err != nil {
		t.Fatalf("cavity cut: %v", err)
	}
	return res
}

// rotZ45 is a 45° rotation about the Z axis through the origin — used to make a block's
// world AABB strictly larger than the block, so an envelope volume is a meaningful gate.
func rotZ45(t *testing.T) math.Matrix4 {
	t.Helper()
	z, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("NewUnitVector3: %v", err)
	}
	return math.Rotation4(stdmath.Pi/4, z, math.P3(0, 0, 0))
}

// TestShrinkwrapMergesAllPartsByDefault: the zero-value recipe (keep all, no envelope)
// merges every placed body into one base, volume = sum (two disjoint 2³ blocks → 16).
func TestShrinkwrapMergesAllPartsByDefault(t *testing.T) {
	t.Parallel()
	block := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: math.Identity4(), Source: occFor("a:1")},
		{Body: block, Transform: math.Translation4(math.V3(10, 0, 0)), Source: occFor("a:2")},
	}}
	bodies, err := BuildShrinkwrap(src, ShrinkwrapDefinition{})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("BuildShrinkwrap: err=%v bodies=%d, want one base body", err, len(bodies))
	}
	if got := volumeOf(bodies[0]); !approx(got, 16) {
		t.Errorf("shrinkwrap volume = %g, want 16 (two 2³ blocks merged)", got)
	}
}

// TestShrinkwrapRemovesSmallParts drops a tiny part below the size threshold, keeping
// only the large one (volume-gated against the kept block's analytic volume).
func TestShrinkwrapRemovesSmallParts(t *testing.T) {
	t.Parallel()
	big := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))              // volume 8
	tiny := solidBlock(t, math.P3(10, 10, 10), math.P3(10.5, 10.5, 10.5)) // volume 0.125
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: big, Transform: math.Identity4(), Source: occFor("big:1")},
		{Body: tiny, Transform: math.Identity4(), Source: occFor("tiny:1")},
	}}
	bodies, err := BuildShrinkwrap(src, ShrinkwrapDefinition{RemoveStyle: RemoveSmallParts, MinPartVolume: 1})
	if err != nil {
		t.Fatalf("BuildShrinkwrap: %v", err)
	}
	if got := volumeOf(bodies[0]); !approx(got, 8) {
		t.Errorf("shrinkwrap volume = %g, want 8 (0.125³ part removed by size)", got)
	}
}

// TestShrinkwrapRemovesInternalParts drops a part fully enclosed by another. The small
// block sits entirely inside the big one; without removal a multi-lump merge would
// double-count it (65), so the analytic 64 proves it was dropped.
func TestShrinkwrapRemovesInternalParts(t *testing.T) {
	t.Parallel()
	big := solidBlock(t, math.P3(0, 0, 0), math.P3(4, 4, 4))   // volume 64
	inner := solidBlock(t, math.P3(1, 1, 1), math.P3(2, 2, 2)) // volume 1, inside big
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: big, Transform: math.Identity4(), Source: occFor("shell:1")},
		{Body: inner, Transform: math.Identity4(), Source: occFor("core:1")},
	}}
	bodies, err := BuildShrinkwrap(src, ShrinkwrapDefinition{RemoveStyle: RemoveInternalParts})
	if err != nil {
		t.Fatalf("BuildShrinkwrap: %v", err)
	}
	if got := volumeOf(bodies[0]); !approx(got, 64) {
		t.Errorf("shrinkwrap volume = %g, want 64 (enclosed inner part removed)", got)
	}
}

// TestShrinkwrapPerPartEnvelope replaces a part with its bounding box. A 2³ block
// rotated 45° about Z has an axis-aligned envelope of (2√2)²·2 = 16 — twice the block's
// volume — so the envelope clearly changed the geometry.
func TestShrinkwrapPerPartEnvelope(t *testing.T) {
	t.Parallel()
	block := solidBlock(t, math.P3(-1, -1, -1), math.P3(1, 1, 1)) // volume 8
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: rotZ45(t), Source: occFor("a:1")},
	}}
	bodies, err := BuildShrinkwrap(src, ShrinkwrapDefinition{EnvelopeStyle: EnvelopePerPart})
	if err != nil {
		t.Fatalf("BuildShrinkwrap: %v", err)
	}
	if got := volumeOf(bodies[0]); !approx(got, 16) {
		t.Errorf("per-part envelope volume = %g, want 16 (AABB of the rotated 2³ block)", got)
	}
}

// TestShrinkwrapWholeEnvelope replaces the entire assembly with one bounding box: two
// unit blocks 3 apart give an envelope of [0,4]×[0,1]×[0,1] = 4.
func TestShrinkwrapWholeEnvelope(t *testing.T) {
	t.Parallel()
	near := solidBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	far := solidBlock(t, math.P3(3, 0, 0), math.P3(4, 1, 1))
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: near, Transform: math.Identity4(), Source: occFor("a:1")},
		{Body: far, Transform: math.Identity4(), Source: occFor("a:2")},
	}}
	bodies, err := BuildShrinkwrap(src, ShrinkwrapDefinition{EnvelopeStyle: EnvelopeWhole})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("BuildShrinkwrap: err=%v bodies=%d, want one envelope", err, len(bodies))
	}
	if got := volumeOf(bodies[0]); !approx(got, 4) {
		t.Errorf("whole envelope volume = %g, want 4 (4×1×1 bounding box)", got)
	}
}

// TestShrinkwrapPatchHolesFillsCavity gates the hole-patch through the feature: a
// hollow 56-volume part is solidified to its 64-volume outer block when PatchHoles is
// set (the internal void is dropped before merge).
func TestShrinkwrapPatchHolesFillsCavity(t *testing.T) {
	t.Parallel()
	cav := cavityBlock(t)
	if v := volumeOf(cav); v < 55.9 || v > 56.1 {
		t.Fatalf("cavity fixture volume = %g, want ~56", v)
	}
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: cav, Transform: math.Identity4(), Source: occFor("hollow:1")},
	}}
	bodies, err := BuildShrinkwrap(src, ShrinkwrapDefinition{PatchHoles: true})
	if err != nil {
		t.Fatalf("BuildShrinkwrap: %v", err)
	}
	if got := volumeOf(bodies[0]); got < 63.9 || got > 64.1 {
		t.Errorf("patched volume = %g, want ~64 (cavity filled)", got)
	}
}

// holedPlate returns a 4×4×2 plate with a 2×2 through-hole (vol 24), built by extruding a
// rectangle-with-rectangular-hole profile.
func holedPlate(t *testing.T) *topo.Body {
	t.Helper()
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	addRect(sk, 0, 0, 4, 4)
	addRect(sk, 1, 1, 3, 3)
	fs := NewPartFeatures(nil)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 2 })
	fs.Recompute()
	return fs.Result()[0]
}

// TestShrinkwrapCapsThroughHole gates the through-hole patch (#721) through the feature: a
// holed plate (vol 24) closes to its 32-volume solid block when MaxHoleDiameter covers the
// opening, and is left intact when it does not.
func TestShrinkwrapCapsThroughHole(t *testing.T) {
	t.Parallel()
	plate := holedPlate(t)
	if v := volumeOf(plate); v < 23.9 || v > 24.1 {
		t.Fatalf("holed-plate fixture volume = %g, want ~24", v)
	}
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: plate, Transform: math.Identity4(), Source: occFor("plate:1")},
	}}

	capped, err := BuildShrinkwrap(src, ShrinkwrapDefinition{MaxHoleDiameter: 3})
	if err != nil {
		t.Fatalf("BuildShrinkwrap: %v", err)
	}
	if got := volumeOf(capped[0]); got < 31.9 || got > 32.1 {
		t.Errorf("capped volume = %g, want ~32 (hole closed flush)", got)
	}

	// PatchHoles alone must NOT fill a through-hole (it is not a disconnected internal void).
	intact, err := BuildShrinkwrap(src, ShrinkwrapDefinition{PatchHoles: true})
	if err != nil {
		t.Fatalf("BuildShrinkwrap: %v", err)
	}
	if got := volumeOf(intact[0]); got < 23.9 || got > 24.1 {
		t.Errorf("PatchHoles volume = %g, want ~24 (through-hole left intact)", got)
	}
}

// TestShrinkwrapBreakLinkFreezesAndVersionTracks covers the associative pull (a source
// edit re-simplifies) and break-link (the result freezes).
func TestShrinkwrapBreakLinkFreezesAndVersionTracks(t *testing.T) {
	t.Parallel()
	block := solidBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	src := &fakeAssemblySource{placed: []PlacedBody{
		{Body: block, Transform: math.Identity4(), Source: occFor("a:1")},
	}}
	fs := NewPartFeatures(nil)
	pf := NewShrinkwrapComponents(fs).AddShrinkwrap(src, ShrinkwrapDefinition{}, DeriveSourceLink{})
	fs.Recompute()
	s := pf.Definition().(*ShrinkwrapComponent)
	v0 := s.SourceVersion()

	// Edit the source → associative re-simplify reflects the added body.
	src.placed = append(src.placed, PlacedBody{Body: block, Transform: math.Translation4(math.V3(10, 0, 0)), Source: occFor("a:2")})
	src.version++
	fs.MarkDirty(pf)
	fs.Recompute()
	if got := volumeOf(fs.Result()[0]); !approx(got, 16) {
		t.Fatalf("after source edit, volume = %g, want 16 (re-simplified)", got)
	}
	if s.SourceVersion() == v0 {
		t.Error("source version did not advance on edit")
	}

	// Break the link and edit again → frozen, unchanged.
	if err := s.BreakLink(); err != nil {
		t.Fatalf("BreakLink: %v", err)
	}
	if s.Linked() {
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
