// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
)

func TestMoveTranslatesRunningBody(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
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
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(prismBody())
	bad := NewModifyFeatures(fs).AddMove(7, math.Identity4())
	fs.Recompute()
	if bad.Health().Status != health.Sick {
		t.Errorf("move with bad index = %v, want sick", bad.Health().Status)
	}
}

func TestMoveFeatureRoundTrip(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	tx := math.Rotation4(0.5, math.V3(0, 0, 1).AsUnit(), math.P3(0, 0, 0)).Mul(math.Translation4(math.V3(2, -3, 4)))
	NewModifyFeatures(fs).AddMove(1, tx)

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil, nil)
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
