// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Face fillet (#694, adjacent-faces case): round the edge(s) shared by two face sets, selecting by
// face instead of by edge.

// boxAndPlanarFace builds a 2×2×2 box and returns the engine plus a helper that finds the reference
// key of the axis-aligned planar face flush against the given min/max plane (e.g. top = z==2).
func boxAndPlanarFace(t *testing.T) (*PartFeatures, func(axis byte, at float64) []byte) {
	t.Helper()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	find := func(axis byte, at float64) []byte {
		t.Helper()
		for _, f := range box.Faces() {
			rb := f.RangeBox()
			lo, hi := boxAxisSpan(rb, axis)
			if lo == at && hi == at {
				return f.ReferenceKey()
			}
		}
		t.Fatalf("no planar face flush at %c==%g", axis, at)
		return nil
	}
	return fs, find
}

func boxAxisSpan(rb math.Box, axis byte) (float64, float64) {
	switch axis {
	case 'x':
		return rb.Min.X, rb.Max.X
	case 'y':
		return rb.Min.Y, rb.Max.Y
	default:
		return rb.Min.Z, rb.Max.Z
	}
}

// TestFaceFilletRoundsSharedEdge rounds the single edge shared by the top face and a side face,
// selected by face: the result is a valid solid whose volume is the box minus one quarter-round.
func TestFaceFilletRoundsSharedEdge(t *testing.T) {
	fs, face := boxAndPlanarFace(t)
	top := face('z', 2)
	side := face('x', 2)
	pf := NewDressUpFeatures(fs).AddFaceFillet([][]byte{top}, [][]byte{side}, func() float64 { return 0.3 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("face fillet sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("face fillet not a valid solid: %+v", r)
	}
	if !hasCylinderFace(res) {
		t.Error("face fillet produced no cylindrical face")
	}
	notch := func(r float64) float64 { return r*r - stdmath.Pi*r*r/4 } // cross-section removed per unit length
	want := 8 - notch(0.3)*2                                           // one edge of length 2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; relErr(got, want) > 1e-3 {
		t.Errorf("face fillet volume = %g, want ≈ %g", got, want)
	}
}

// TestFaceFilletNonAdjacentHealsAndRounds: a 4×4×4 box with a chamfered vertical edge — the +X and
// +Y faces share no edge (the chamfer face separates them). The non-adjacent face fillet heals the
// chamfer to the virtual edge and rounds it: a valid solid whose volume is the FULL box minus one
// quarter-round (the chamfer absorbed), r=1 over a length-4 edge (#694).
func TestFaceFilletNonAdjacentHealsAndRounds(t *testing.T) {
	pent := []math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 3}, {X: 3, Y: 4}, {X: 0, Y: 4}}
	box := buildPrism(pent, sketch.XYPlane(), span{near: 0, far: 4}, 0, "box")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)

	xf := faceKeyByNormal(t, box, math.V3(1, 0, 0))
	yf := faceKeyByNormal(t, box, math.V3(0, 1, 0))
	pf := NewDressUpFeatures(fs).AddFaceFillet([][]byte{xf}, [][]byte{yf}, func() float64 { return 1 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("non-adjacent face fillet sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("non-adjacent face fillet not a valid solid: %+v", r)
	}
	if !hasCylinderFace(res) {
		t.Error("non-adjacent face fillet produced no cylindrical face")
	}
	want := 64 - (1*1-stdmath.Pi/4)*4 // full box minus the quarter-round notch, length 4
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; relErr(got, want) > 2e-3 {
		t.Errorf("non-adjacent face fillet volume = %g, want ≈ %g", got, want)
	}
}

// faceKeyByNormal returns the reference key of the body's planar face whose normal points along n.
func faceKeyByNormal(t *testing.T, b *topo.Body, n math.Vector3) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok && float64(pl.Normal().Dot(n)) > 0.99 {
			return f.ReferenceKey()
		}
	}
	t.Fatalf("no planar face with normal %v", n)
	return nil
}

// TestFaceFilletNonAdjacentSick: two parallel faces that share no edge (top and bottom) cannot be
// healed to a virtual edge (their planes never meet) — the feature goes Sick rather than silently
// doing nothing.
func TestFaceFilletNonAdjacentSick(t *testing.T) {
	fs, face := boxAndPlanarFace(t)
	pf := NewDressUpFeatures(fs).AddFaceFillet([][]byte{face('z', 2)}, [][]byte{face('z', 0)}, func() float64 { return 0.3 })
	fs.Recompute()
	if pf.Health().OK() {
		t.Fatal("face fillet of two non-adjacent faces should be sick (no shared edge)")
	}
}

func hasCylinderFace(b *topo.Body) bool {
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return true
		}
	}
	return false
}
