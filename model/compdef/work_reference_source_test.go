// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

const refEps = 1e-9

// TestWorkPointRefSourceResolvesOriginCenter: the origin centre point projects to (0,0,0).
func TestWorkPointRefSourceResolvesOriginCenter(t *testing.T) {
	d := NewPartComponentDefinition()
	src := NewWorkPointRefSource(d, feature.OriginCenter)
	if src.SourceID() != string(feature.OriginCenter) {
		t.Errorf("SourceID = %q, want %q", src.SourceID(), feature.OriginCenter)
	}
	pos, ok := src.Position()
	if !ok {
		t.Fatal("origin centre point should resolve")
	}
	if !pos.IsEqualTo(math.P3(0, 0, 0), refEps) {
		t.Errorf("origin centre = %v, want (0,0,0)", pos)
	}
}

// TestWorkPointRefSourceLostReference: an unknown datum reference reports lost, not a panic.
func TestWorkPointRefSourceLostReference(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, ok := NewWorkPointRefSource(d, "origin/point/bogus").Position(); ok {
		t.Error("unknown datum point should report ok=false")
	}
}

// TestWorkAxisRefSourceSamplesAlongAxis: the origin X axis samples a segment centred on the
// origin and directed along +X, sized by the default half-span on an empty part.
func TestWorkAxisRefSourceSamplesAlongAxis(t *testing.T) {
	d := NewPartComponentDefinition()
	pts, ok := NewWorkAxisRefSource(d, feature.OriginXAxis).SamplePoints()
	if !ok || len(pts) != 2 {
		t.Fatalf("X axis sample = %v (ok=%v), want two endpoints", pts, ok)
	}
	wantLo, wantHi := math.P3(-referenceLineHalfSpan, 0, 0), math.P3(referenceLineHalfSpan, 0, 0)
	if !pts[0].IsEqualTo(wantLo, refEps) || !pts[1].IsEqualTo(wantHi, refEps) {
		t.Errorf("X axis endpoints = %v..%v, want %v..%v", pts[0], pts[1], wantLo, wantHi)
	}
}

// TestWorkPlaneRefSourceIntersectsSketchPlane: projecting the origin XZ plane onto an XY sketch
// yields the X axis line (the planes meet along y=0, z=0).
func TestWorkPlaneRefSourceIntersectsSketchPlane(t *testing.T) {
	d := NewPartComponentDefinition()
	pts, ok := NewWorkPlaneRefSource(d, feature.OriginXZPlane, sketch.XYPlane()).SamplePoints()
	if !ok || len(pts) != 2 {
		t.Fatalf("XZ∩XY sample = %v (ok=%v), want two endpoints", pts, ok)
	}
	for _, p := range pts {
		if !math.IsNearZero(p.Y, refEps) || !math.IsNearZero(p.Z, refEps) {
			t.Errorf("intersection point %v not on the X axis (y=z=0)", p)
		}
	}
	if math.IsNearZero(p2pSpanX(pts), refEps) {
		t.Error("intersection line has no extent along X")
	}
}

// TestWorkPlaneRefSourceParallelHasNoLine: an XY work plane projected onto an XY sketch is
// parallel — there is no intersection line, so the source reports lost.
func TestWorkPlaneRefSourceParallelHasNoLine(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, ok := NewWorkPlaneRefSource(d, feature.OriginXYPlane, sketch.XYPlane()).SamplePoints(); ok {
		t.Error("parallel planes should report ok=false (no intersection line)")
	}
}

// TestReferenceLineHalfSpanDefaultsOnEmptyPart: with no geometry the span is the default.
func TestReferenceLineHalfSpanDefaultsOnEmptyPart(t *testing.T) {
	if got := NewPartComponentDefinition().referenceLineHalfSpan(); got != referenceLineHalfSpan {
		t.Errorf("empty-part span = %v, want %v", got, referenceLineHalfSpan)
	}
}

// TestWorkKeyResolvesClassifiesDatums: the resolves helpers classify each origin reference and
// reject a B-rep-style key.
func TestWorkKeyResolvesClassifiesDatums(t *testing.T) {
	d := NewPartComponentDefinition()
	if !d.WorkPointKeyResolves(string(feature.OriginCenter)) {
		t.Error("origin centre should resolve as a work point")
	}
	if !d.WorkAxisKeyResolves(string(feature.OriginXAxis)) {
		t.Error("origin X axis should resolve as a work axis")
	}
	if !d.WorkPlaneKeyResolves(string(feature.OriginXYPlane)) {
		t.Error("origin XY plane should resolve as a work plane")
	}
	if d.WorkPointKeyResolves("not-a-datum") {
		t.Error("a non-datum reference must not resolve as a work point")
	}
}

// TestWorkAxisAndPlaneSourceID: the axis/plane sources report their datum reference.
func TestWorkAxisAndPlaneSourceID(t *testing.T) {
	d := NewPartComponentDefinition()
	if got := NewWorkAxisRefSource(d, feature.OriginXAxis).SourceID(); got != string(feature.OriginXAxis) {
		t.Errorf("axis SourceID = %q, want %q", got, feature.OriginXAxis)
	}
	if got := NewWorkPlaneRefSource(d, feature.OriginXZPlane, sketch.XYPlane()).SourceID(); got != string(feature.OriginXZPlane) {
		t.Errorf("plane SourceID = %q, want %q", got, feature.OriginXZPlane)
	}
}

// TestWorkAxisRefSourceLostReference / TestWorkPlaneRefSourceLostReference: an unknown datum
// reference reports lost rather than panicking.
func TestWorkAxisRefSourceLostReference(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, ok := NewWorkAxisRefSource(d, "origin/axis/bogus").SamplePoints(); ok {
		t.Error("unknown datum axis should report ok=false")
	}
}

func TestWorkPlaneRefSourceLostReference(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, ok := NewWorkPlaneRefSource(d, "origin/plane/bogus", sketch.XYPlane()).SamplePoints(); ok {
		t.Error("unknown datum plane should report ok=false")
	}
}

// TestWorkPlaneIntersectsSketch: the viability probe is true for a meeting plane, false for a
// parallel one.
func TestWorkPlaneIntersectsSketch(t *testing.T) {
	d := NewPartComponentDefinition()
	if !d.WorkPlaneIntersectsSketch(feature.OriginXZPlane, sketch.XYPlane()) {
		t.Error("XZ plane should meet the XY sketch in a line")
	}
	if d.WorkPlaneIntersectsSketch(feature.OriginXYPlane, sketch.XYPlane()) {
		t.Error("XY plane is parallel to the XY sketch — no intersection line")
	}
}

// TestReferenceLineHalfSpanScalesWithModel: a large body grows the reference-line span beyond
// the empty-part default (half the model range-box diagonal).
func TestReferenceLineHalfSpanScalesWithModel(t *testing.T) {
	d := NewPartComponentDefinition()
	sk := d.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(100, 0))
	c2 := sk.Points().Add(math.P2(100, 100))
	c3 := sk.Points().Add(math.P2(0, 100))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(d.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 100 })
	d.Recompute()
	if got := d.referenceLineHalfSpan(); got <= referenceLineHalfSpan {
		t.Errorf("large-model span = %v, want > default %v", got, referenceLineHalfSpan)
	}
}

// TestReferenceLineHalfSpanSmallModelKeepsDefault: a tiny body (half-diagonal below the
// default) keeps the default span so the reference line stays visible.
func TestReferenceLineHalfSpanSmallModelKeepsDefault(t *testing.T) {
	d := NewPartComponentDefinition()
	sk := d.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(1, 0))
	c2 := sk.Points().Add(math.P2(1, 1))
	c3 := sk.Points().Add(math.P2(0, 1))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(d.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 1 })
	d.Recompute()
	if got := d.referenceLineHalfSpan(); got != referenceLineHalfSpan {
		t.Errorf("small-model span = %v, want default %v", got, referenceLineHalfSpan)
	}
}

func p2pSpanX(pts []math.Point3) float64 { return pts[1].X - pts[0].X }
