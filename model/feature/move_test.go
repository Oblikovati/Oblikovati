// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
)

func TestMoveTranslatesRunningBody(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody()) // unit prism at the origin
	move := NewModifyFeatures(fs).AddMove(0, math.Translation4(math.V3(5, 0, 0)))
	fs.Recompute()

	if !move.Health().OK() {
		t.Fatalf("move sick: %+v", move.Health())
	}
	if len(fs.Result()) != 1 {
		t.Fatalf("move result = %d bodies, want 1", len(fs.Result()))
	}
	got := fs.Result()[0]
	if r := ops.Validate(got); !r.Valid {
		t.Fatalf("moved body invalid: %v", r.Issues)
	}
	if box := got.RangeBox(); stdmath.Abs(box.Min.X-5) > 1e-9 || stdmath.Abs(box.Max.X-6) > 1e-9 {
		t.Errorf("moved range box X = [%g,%g], want [5,6]", box.Min.X, box.Max.X)
	}
}

func TestMoveRejectsBadIndex(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	bad := NewModifyFeatures(fs).AddMove(7, math.Identity4())
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Errorf("move with bad index = %v, want sick", bad.Health().Status)
	}
}

func TestMoveFeatureRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	tx := math.Rotation4(0.5, math.V3(0, 0, 1).AsUnit(), math.P3(0, 0, 0)).Mul(math.Translation4(math.V3(2, -3, 4)))
	NewModifyFeatures(fs).AddMove(1, tx)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if fresh.Count() != 1 {
		t.Fatalf("feature count after round trip = %d, want 1", fresh.Count())
	}
	def := fresh.Item(0).Definition().(*MoveFeature).Definition()
	if def.BodyIndex != 1 {
		t.Errorf("body index = %d, want 1", def.BodyIndex)
	}
	if !def.Transform.IsEqualTo(tx, 1e-12) {
		t.Errorf("transform not preserved:\n%v\n%v", def.Transform.Cells(), tx.Cells())
	}
}

// TestMoveOpsComposeInOrder drives the unit prism with "rotate 90° about Z, then slide
// +X 5": the rotation maps x∈[0,1] to x∈[-1,0], and the later slide carries it to [4,5],
// proving the operations compose in list order (M20-F20, #654).
func TestMoveOpsComposeInOrder(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	ops := []MoveOperation{
		RotateAboutLineOp(math.P3(0, 0, 0), math.V3(0, 0, 1), constFloat(stdmath.Pi/2)),
		AlongRayOp(math.V3(1, 0, 0), constFloat(5)),
	}
	move := NewModifyFeatures(fs).AddMoveOps(0, ops)
	fs.Recompute()

	if !move.Health().OK() {
		t.Fatalf("move sick: %+v", move.Health())
	}
	box := fs.Result()[0].RangeBox()
	if stdmath.Abs(box.Min.X-4) > 1e-9 || stdmath.Abs(box.Max.X-5) > 1e-9 {
		t.Errorf("composed range box X = [%g,%g], want [4,5]", box.Min.X, box.Max.X)
	}
	if stdmath.Abs(box.Min.Y-0) > 1e-9 || stdmath.Abs(box.Max.Y-1) > 1e-9 {
		t.Errorf("composed range box Y = [%g,%g], want [0,1]", box.Min.Y, box.Max.Y)
	}
}

// TestMoveOpsRecomputeOnParamChange proves an operation's scalar is re-read live: moving
// the closure's backing value and recomputing relocates the body to the new distance.
func TestMoveOpsRecomputeOnParamChange(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	dist := 2.0
	move := NewModifyFeatures(fs).AddMoveOps(0, []MoveOperation{
		AlongRayOp(math.V3(1, 0, 0), func() float64 { return dist }),
	})
	fs.Recompute()
	if box := fs.Result()[0].RangeBox(); stdmath.Abs(box.Min.X-2) > 1e-9 {
		t.Fatalf("range box X min = %g, want 2 before edit", box.Min.X)
	}
	// A parameter edit marks dependent features dirty; here we change the backing value
	// and mark the move so the next recompute re-reads the operation's live closure.
	dist = 7
	fs.MarkDirty(move)
	fs.Recompute()
	if box := fs.Result()[0].RangeBox(); stdmath.Abs(box.Min.X-7) > 1e-9 {
		t.Errorf("range box X min = %g, want 7 after param change", box.Min.X)
	}
}

// TestMoveOpsRoundTrip serializes and restores an operation list and checks the operation
// kinds and the composed transform survive (M20-F20).
func TestMoveOpsRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	orig := []MoveOperation{
		RotateAboutLineOp(math.P3(1, 0, 0), math.V3(0, 0, 1), constFloat(0.4)),
		AlongRayOp(math.V3(0, 1, 0), constFloat(3)),
		FreeDragOp(constFloat(1), constFloat(-2), constFloat(0.5)),
	}
	NewModifyFeatures(fs).AddMoveOps(2, orig)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*MoveFeature).Definition()
	if def.BodyIndex != 2 || len(def.Ops) != 3 {
		t.Fatalf("restored move = body %d, %d ops; want body 2, 3 ops", def.BodyIndex, len(def.Ops))
	}
	wantKinds := []types.MoveOperationType{types.MoveRotateAboutLine, types.MoveAlongRay, types.MoveFreeDrag}
	for i, want := range wantKinds {
		if def.Ops[i].Kind != want {
			t.Errorf("op %d kind = %q, want %q", i, def.Ops[i].Kind, want)
		}
	}
	if want := composeMoveOps(orig); !def.transform().IsEqualTo(want, 1e-9) {
		t.Errorf("composed transform not preserved:\n%v\n%v", def.transform().Cells(), want.Cells())
	}
}
