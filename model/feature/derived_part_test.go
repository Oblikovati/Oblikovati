// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// offsetBlockSource is a fake part source holding a unit block offset to +x (centroid
// x=1.5), so a reflection about x=0 visibly relocates it to -x while preserving volume.
func offsetBlockSource(t *testing.T) *fakeBodySource {
	t.Helper()
	src := newFakeBodySource()
	src.SurfaceBodies().Add(solidBlock(t, math.P3(1, 0, 0), math.P3(2, 1, 1)))
	return src
}

// TestDerivedPartAppliesReflection checks a reflection (negative-determinant) derive
// transform mirrors the source body — the volume is preserved (the winding flip keeps it
// a valid outward solid, not an inside-out negative one) and the centroid crosses to -x.
func TestDerivedPartAppliesReflection(t *testing.T) {
	src := offsetBlockSource(t)
	reflect := math.Reflection4(math.P3(0, 0, 0), mustX()) // mirror across the x=0 plane

	fs := NewPartFeatures(nil, nil)
	pf := NewDerivedComponents(fs).AddDerived(src, reflect, DeriveSourceLink{})
	fs.Recompute()
	if !pf.Health().OK() || len(fs.Result()) != 1 {
		t.Fatalf("derived reflection: health=%+v bodies=%d, want ok and one body", pf.Health(), len(fs.Result()))
	}
	props := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality())
	if !approx(props.Volume, 1) {
		t.Errorf("mirrored volume = %g, want 1 (reflection preserves volume; winding stays outward)", props.Volume)
	}
	if props.Centroid.X > -1 {
		t.Errorf("mirrored centroid x = %g, want ≈ -1.5 (reflected across x=0)", props.Centroid.X)
	}
}

// TestDerivedPartRoundTripAndRebind round-trips a reflecting derive through the feature
// codec (link + transform + linked), restores it UNBOUND, rebinds a newer-revision source,
// and checks it flags out of date and re-derives the mirrored body.
func TestDerivedPartRoundTripAndRebind(t *testing.T) {
	src := offsetBlockSource(t)
	reflect := math.Reflection4(math.P3(0, 0, 0), mustX())
	link := DeriveSourceLink{Document: "src.obk", InternalName: "GUID-1", DatabaseRevisionID: "rev-1"}
	fs := NewPartFeatures(nil, nil)
	pf := NewDerivedComponents(fs).AddDerived(src, reflect, link)

	fd, err := serializeFeature(pf, nil, nil)
	if err != nil {
		t.Fatalf("serialize derived part: %v", err)
	}
	fs2 := NewPartFeatures(nil, nil)
	restored, err := buildFeature(fs2, fd, nil, nil, nil)
	if err != nil {
		t.Fatalf("restore derived part: %v", err)
	}
	d := restored.Definition().(*DerivedPartComponent)
	if d.SourceLink() != link {
		t.Errorf("restored link = %+v, want %+v", d.SourceLink(), link)
	}
	if d.Transform().Cells() != reflect.Cells() {
		t.Error("restored transform does not match the saved reflection")
	}
	if d.SourceVersion() != "" {
		t.Errorf("restored derive should be unbound, got source version %q", d.SourceVersion())
	}

	d.BindSource(offsetBlockSource(t), "rev-2")
	if !d.OutOfDate() {
		t.Error("rebinding a newer-revision source should flag out of date")
	}
	fs2.Recompute()
	if len(fs2.Result()) != 1 || !approx(volumeOf(fs2.Result()[0]), 1) {
		t.Fatalf("rebound derive = %v, want one mirrored body of volume 1", fs2.Result())
	}
}
