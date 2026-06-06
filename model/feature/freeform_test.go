// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
)

func TestFreeformPrimitiveConvertsToBRep(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewFreeformFeatures(fs).AddBox(2, 2, 2, 0)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("freeform went unhealthy: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if !body.IsSolid() {
		t.Error("a free-form box primitive should convert to a solid B-rep")
	}
	if r := ops.Validate(body); !r.Valid {
		t.Errorf("free-form body failed validation: %+v", r)
	}
}

func TestFreeformCageEditDeformsBody(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewFreeformFeatures(fs).AddBox(2, 2, 2, 1)
	fs.Recompute()
	before := fs.Result()[0].RangeBox()
	// Pull cage vertex 6 (the +X+Y+Z corner) outward; the limit surface follows.
	ff := pf.Definition().(*FreeformFeature)
	ff.FreeformBody().MoveVertices([]int{6}, math.V3(3, 3, 3))
	fs.MarkDirty(fs.Item(0))
	fs.Recompute()
	after := fs.Result()[0].RangeBox()
	if !(after.Max.X > before.Max.X && after.Max.Y > before.Max.Y && after.Max.Z > before.Max.Z) {
		t.Errorf("moving a cage vertex should grow the body: before %v, after %v", before.Max, after.Max)
	}
}

func TestFreeformCreaseSharpensCorner(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewFreeformFeatures(fs).AddBox(2, 2, 2, 3)
	ff := pf.Definition().(*FreeformFeature)
	// Crease the three edges meeting at corner 0 → it stays a sharp point at (0,0,0).
	ff.FreeformBody().CreaseEdges([][2]int{{0, 1}, {0, 3}, {0, 4}}, 1)
	fs.Recompute()
	// The creased corner must be present in the converted body's range box minimum.
	box := fs.Result()[0].RangeBox()
	if !approxEq(box.Min.X, 0) || !approxEq(box.Min.Y, 0) || !approxEq(box.Min.Z, 0) {
		t.Errorf("creased corner not preserved: range min = %v, want (0,0,0)", box.Min)
	}
}

func TestFreeformQuadBallIsRounded(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewFreeformFeatures(fs).AddQuadBall(5, 0)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("quad-ball went unhealthy: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if !body.IsSolid() {
		t.Error("a quad-ball should be a closed solid")
	}
	// Its extent is roughly the diameter (rounded, so a bit under 10).
	if d := body.RangeBox().Diagonal(); d.X > 10.001 || d.X < 6 {
		t.Errorf("quad-ball x-extent = %v, want ~ (6,10)", d.X)
	}
}

func TestFreeformEdgeAndVertexHandles(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	pf := NewFreeformFeatures(fs).AddBox(2, 2, 2, 0)
	ff := pf.Definition().(*FreeformFeature)
	b := ff.FreeformBody()
	if b.Vertices().Count() != 8 {
		t.Errorf("box cage has %d vertices, want 8", b.Vertices().Count())
	}
	if b.Edges().Count() != 12 {
		t.Errorf("box cage has %d edges, want 12", b.Edges().Count())
	}
	if b.Faces().Count() != 6 {
		t.Errorf("box cage has %d faces, want 6", b.Faces().Count())
	}
	// A vertex handle reads then moves its position.
	v := b.Vertices().Item(0)
	v.Move(math.V3(1, 0, 0))
	if !approxEq(b.Vertices().Item(0).Point().X, 1) {
		t.Errorf("vertex move not reflected: x = %v, want 1", b.Vertices().Item(0).Point().X)
	}
}

func TestAliasFreeformWrapsImportedCage(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	pf := NewAliasFreeformFeatures(fs).AddFromCage(verts, [][]int{{0, 1, 2, 3}}, 0)
	fs.Recompute()
	if pf.Kind() != "alias-freeform" {
		t.Errorf("kind = %q, want alias-freeform", pf.Kind())
	}
	if !pf.Health().OK() || len(fs.Result()) != 1 {
		t.Errorf("alias free-form did not produce a body: health %+v, bodies %d", pf.Health(), len(fs.Result()))
	}
}
