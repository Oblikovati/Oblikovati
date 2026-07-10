// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// TestOccurrenceDatumTransform: a sub-component's datums, named through an occurrence-qualified ref,
// resolve into the assembly's space with the occurrence's placement transform applied — the core of
// #1857 (both the AC3 input path via ResolvePlaneRef/ResolvePointRef, and the AC1 context helper).
func TestOccurrenceDatumTransform(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	part := NewPartComponentDefinition()
	wp := part.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 5 }) // z = 5 in part space
	pt := part.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 2, 3) })
	part.Recompute()
	asm.Place("sub:1", part, math.Translation4(math.V3(10, 0, 0))) // placed +10 along X

	xf, work, ok := asm.ResolveOccurrenceContext([]string{"sub:1"})
	if !ok || work == nil {
		t.Fatal("occurrence context should resolve to the placed part's work geometry")
	}
	if !xf.TransformPoint(math.P3(0, 0, 0)).IsEqualTo(math.P3(10, 0, 0), 1e-9) {
		t.Errorf("occurrence transform of origin = %v, want (10,0,0)", xf.TransformPoint(math.P3(0, 0, 0)))
	}

	// AC3: the assembly's own resolver accepts the occ-qualified ref as a datum input, transformed.
	planeRef := feature.EncodeOccurrenceRef([]string{"sub:1"}, wp.Key())
	pl, err := asm.WorkGeometry().ResolvePlaneRef(planeRef)
	if err != nil {
		t.Fatalf("resolve occurrence plane: %v", err)
	}
	if !pl.Origin().IsEqualTo(math.P3(10, 0, 5), 1e-9) {
		t.Errorf("occurrence plane origin = %v, want (10,0,5)", pl.Origin())
	}
	if !pl.Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), 1e-9) {
		t.Errorf("occurrence plane normal = %v, want +Z (rigid translation preserves it)", pl.Normal())
	}

	pointRef := feature.EncodeOccurrenceRef([]string{"sub:1"}, pt.Key())
	p, err := asm.WorkGeometry().ResolvePointRef(pointRef)
	if err != nil {
		t.Fatalf("resolve occurrence point: %v", err)
	}
	if !p.IsEqualTo(math.P3(11, 2, 3), 1e-9) {
		t.Errorf("occurrence point = %v, want (11,2,3)", p)
	}

	// The axis resolver transforms origin + direction into assembly space (feeds WorkGeometry.axis()).
	dir, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	ax := part.WorkAxes().AddByLine(math.P3(0, 0, 0), dir)
	part.Recompute()
	o, d, ok := occurrenceDatumResolver{asm: asm}.OccurrenceAxisLine(feature.EncodeOccurrenceRef([]string{"sub:1"}, ax.Key()))
	if !ok || !o.IsEqualTo(math.P3(10, 0, 0), 1e-9) || !d.AsVector().IsEqualTo(math.V3(1, 0, 0), 1e-9) {
		t.Errorf("occurrence axis = origin %v dir %v ok %v, want (10,0,0)/+X", o, d, ok)
	}
	if _, _, ok := (occurrenceDatumResolver{asm: asm}).OccurrenceAxisLine("plane/3"); ok {
		t.Error("a non-occurrence ref must not resolve as an occurrence axis")
	}
}

// TestOccurrenceContextUnresolvable: an unknown occurrence path, and a ref whose leaf is not a part,
// both fail cleanly rather than panicking (#1857).
func TestOccurrenceContextUnresolvable(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	if _, _, ok := asm.ResolveOccurrenceContext([]string{"missing:1"}); ok {
		t.Error("an unknown occurrence path should not resolve")
	}
	if _, err := asm.WorkGeometry().ResolvePlaneRef(feature.EncodeOccurrenceRef([]string{"missing:1"}, "plane/3")); err == nil {
		t.Error("an occurrence plane through an unknown path should error")
	}
}
