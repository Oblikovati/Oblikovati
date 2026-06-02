// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"github.com/Oblikovati/oblikovati/model/health"
)

func TestRectangularPatternElementCountFromParameters(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := fs.Add(body()) // a source feature to pattern
	nx, ny := 3, 2
	pat := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()}, func() int { return nx }, func() int { return ny })
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
	// Patterns resolve their source but defer geometry → warning.
	if patPF.Health().Status != health.Warning {
		t.Errorf("pattern health = %v, want warning (geometry deferred)", patPF.Health().Status)
	}
}

func TestPerElementSuppression(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := fs.Add(body())
	pat := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()}, func() int { return 5 }, func() int { return 1 })
	patPF, _ := fs.ByID(patIDOf(fs, pat))
	fs.Recompute()
	if pat.ActiveCount() != 5 {
		t.Fatalf("active = %d, want 5", pat.ActiveCount())
	}
	pat.SetElementSuppressed(2, true)
	fs.MarkDirty(patPF)
	fs.Recompute()
	if pat.ActiveCount() != 4 || !pat.Elements()[2].Suppressed {
		t.Errorf("after suppressing element 2: active=%d, elem2.suppressed=%v", pat.ActiveCount(), pat.Elements()[2].Suppressed)
	}
}

func TestCircularSketchDrivenAndMirror(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := fs.Add(body())
	pats := NewPatternFeatures(fs)
	circ := pats.AddCircular([]ID{src.ID()}, func() int { return 8 }, func() float64 { return 6.28 })
	sk := pats.AddSketchDriven([]ID{src.ID()}, func() int { return 3 })
	mir := pats.AddMirror([]ID{src.ID()}, []byte("plane-key"))
	fs.Recompute()

	if circ.ElementCount() != 8 {
		t.Errorf("circular → %d, want 8", circ.ElementCount())
	}
	if sk.ElementCount() != 3 {
		t.Errorf("sketch-driven → %d, want 3", sk.ElementCount())
	}
	if mir.ElementCount() != 1 {
		t.Errorf("mirror → %d elements, want 1", mir.ElementCount())
	}
	if circ.Kind() != "circular-pattern" || sk.Kind() != "sketch-driven-pattern" || mir.Kind() != "mirror" {
		t.Error("pattern kinds wrong")
	}
	if circ.Definition().Count() != 8 || len(mir.Definition().MirrorPlaneKey) == 0 {
		t.Error("pattern definitions not accessible")
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

func TestPatternAndModifyDefinitionAccessors(t *testing.T) {
	fs := NewPartFeatures(nil, nil)
	src := fs.Add(body())
	rect := NewPatternFeatures(fs).AddRectangular([]ID{src.ID()}, func() int { return 2 }, func() int { return 3 })
	if rect.Definition().CountX() != 2 || rect.Definition().CountY() != 3 {
		t.Error("rectangular definition not accessible")
	}
	sk := NewPatternFeatures(fs).AddSketchDriven([]ID{src.ID()}, func() int { return 4 })
	if sk.Definition().PointCount() != 4 {
		t.Error("sketch-driven definition not accessible")
	}
	// SetElementSuppressed before any recompute (lazy suppressed map init).
	mir := NewPatternFeatures(fs).AddMirror([]ID{src.ID()}, []byte("p"))
	mir.SetElementSuppressed(0, true)
	fs.Recompute()
	if !mir.Elements()[0].Suppressed {
		t.Error("pre-recompute element suppression not applied")
	}
}
