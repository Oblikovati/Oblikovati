// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"sort"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
)

// noStep is a zero grid step for the bookkeeping-only tests (geometry coincides).
var noStep = math.V3(0, 0, 0)

// minXs returns the sorted per-body minimum X of the running result — the placed
// occurrences read out as their grid positions.
func minXs(bodies []*topo.Body) []float64 {
	xs := make([]float64, len(bodies))
	for i, b := range bodies {
		xs[i] = b.RangeBox().Min.X
	}
	sort.Float64s(xs)
	return xs
}

func TestRectangularPatternElementCountFromParameters(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := fs.Add(body()) // a source feature to pattern
	nx, ny := 3, 2
	pat := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()}, func() int { return nx }, func() int { return ny }, noStep, noStep)
	patPF, _ := fs.ByID(patIDOf(fs, pat))
	fs.Recompute()

	if pat.ElementCount() != 6 {
		t.Fatalf("3x2 grid → %d elements, want 6", pat.ElementCount())
	}
	// The element count is parameter-driven: change the counts and re-pattern.
	nx, ny = 4, 4
	fs.MarkDirty(patPF)
	fs.Recompute()
	if pat.ElementCount() != 16 {
		t.Errorf("4x4 grid → %d elements, want 16", pat.ElementCount())
	}
	// Geometry is real now: a pattern that resolves its source is healthy, not deferred.
	if patPF.Health().Status != health.OK {
		t.Errorf("pattern health = %v, want OK (real geometry)", patPF.Health().Status)
	}
}

func TestRectangularPatternPlacesRealCopies(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := NewBaseFeatures(fs).AddBase(prismBody()) // unit cube [0,1]^3
	NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 3 }, func() int { return 1 }, math.V3(2, 0, 0), noStep)
	fs.Recompute()

	res := fs.Result()
	if len(res) != 3 {
		t.Fatalf("1x3 pattern → %d bodies, want 3 placed solids", len(res))
	}
	for i, b := range res {
		if r := ops.Validate(b); !r.Valid {
			t.Fatalf("placed copy %d invalid: %v", i, r.Issues)
		}
	}
	if got := minXs(res); got[0] != 0 || got[1] != 2 || got[2] != 4 {
		t.Errorf("occurrence X positions = %v, want [0 2 4]", got)
	}
}

func TestMirrorReflectsRealCopy(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := NewBaseFeatures(fs).AddBase(prismBody()) // [0,1]^3
	// Mirror across the plane x=0 (normal +X): the copy lands in x∈[-1,0].
	NewPatternFeatures(fs).AddMirror([]ID{src.ID()}, []byte("yz-plane"), math.P3(0, 0, 0), math.V3(1, 0, 0))
	fs.Recompute()

	res := fs.Result()
	if len(res) != 2 {
		t.Fatalf("mirror → %d bodies, want 2 (source + reflected)", len(res))
	}
	mirror := res[1]
	if r := ops.Validate(mirror); !r.Valid {
		t.Fatalf("reflected copy not a valid manifold: %v", r.Issues)
	}
	if box := mirror.RangeBox(); box.Min.X != -1 || box.Max.X != 0 {
		t.Errorf("reflected copy X = [%g,%g], want [-1,0]", box.Min.X, box.Max.X)
	}
}

func TestPerElementSuppressionRemovesOnlyThatCopy(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := NewBaseFeatures(fs).AddBase(prismBody())
	pat := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()},
		func() int { return 3 }, func() int { return 1 }, math.V3(2, 0, 0), noStep)
	patPF, _ := fs.ByID(patIDOf(fs, pat))
	fs.Recompute()
	if pat.ActiveCount() != 3 || len(fs.Result()) != 3 {
		t.Fatalf("active=%d bodies=%d, want 3 and 3", pat.ActiveCount(), len(fs.Result()))
	}
	pat.SetElementSuppressed(1, true) // drop the middle occurrence (x=2)
	fs.MarkDirty(patPF)
	fs.Recompute()
	if pat.ActiveCount() != 2 {
		t.Fatalf("active after suppress = %d, want 2", pat.ActiveCount())
	}
	if got := minXs(fs.Result()); len(got) != 2 || got[0] != 0 || got[1] != 4 {
		t.Errorf("after suppressing element 1: occurrence X = %v, want [0 4]", got)
	}
}

func TestCircularAndSketchDrivenPlaceCopies(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := NewBaseFeatures(fs).AddBase(prismBody())
	pats := NewPatternFeatures(fs)
	circ := pats.AddCircular([]ID{src.ID()}, func() int { return 4 }, func() float64 { return 2 * stdmath.Pi },
		math.P3(0, 0, 0), math.V3(0, 0, 1))
	fs.Recompute()
	if circ.ElementCount() != 4 || len(fs.Result()) != 4 {
		t.Errorf("circular → %d elements / %d bodies, want 4 / 4", circ.ElementCount(), len(fs.Result()))
	}
	for i, b := range fs.Result() {
		if !ops.Validate(b).Valid {
			t.Errorf("circular copy %d invalid", i)
		}
	}

	fs2 := NewPartFeatures(nil, nil)
	src2 := NewBaseFeatures(fs2).AddBase(prismBody())
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(5, 0, 0), math.P3(0, 5, 0)}
	sk := NewPatternFeatures(fs2).AddSketchDriven([]ID{src2.ID()}, func() []math.Point3 { return pts })
	fs2.Recompute()
	if sk.ElementCount() != 3 || len(fs2.Result()) != 3 {
		t.Errorf("sketch-driven → %d elements / %d bodies, want 3 / 3", sk.ElementCount(), len(fs2.Result()))
	}
}

// patIDOf returns the engine id of a pattern feature (it is the last one added of
// its source set — found by identity).
func patIDOf(fs *PartFeatures, f Feature) ID {
	for i := 0; i < fs.Count(); i++ {
		if fs.Item(i).Definition() == f {
			return fs.Item(i).ID()
		}
	}
	return 0
}

func TestPatternDefinitionAccessors(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := fs.Add(body())
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()}, func() int { return 2 }, func() int { return 3 }, noStep, noStep)
	if rect.Definition().CountX() != 2 || rect.Definition().CountY() != 3 {
		t.Error("rectangular definition not accessible")
	}
	pts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}
	sk := NewPatternFeatures(fs).AddSketchDriven([]ID{src.ID()}, func() []math.Point3 { return pts })
	if len(sk.Definition().Points()) != 2 {
		t.Error("sketch-driven definition not accessible")
	}
	mir := NewPatternFeatures(fs).AddMirror([]ID{src.ID()}, []byte("p"), math.P3(0, 0, 0), math.V3(1, 0, 0))
	if len(mir.Definition().MirrorPlaneKey) == 0 {
		t.Error("mirror plane key not accessible")
	}
	mir.SetElementSuppressed(0, true)
	fs.Recompute()
	if !mir.Elements()[0].Suppressed {
		t.Error("pre-recompute element suppression not applied")
	}
}
