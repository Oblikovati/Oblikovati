// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// fakeBodyDef is a body-bearing component definition for the proxy-cut tests: it
// implements occurrence.Definition (RangeBox) and the occurrenceBodies the feature
// reads its source geometry through.
type fakeBodyDef struct{ bodies *topo.SurfaceBodies }

func (d fakeBodyDef) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, b := range d.bodies.All() {
		box = box.Union(b.RangeBox())
	}
	return box
}
func (d fakeBodyDef) SurfaceBodies() *topo.SurfaceBodies { return d.bodies }

// unitBoxDef wraps the solid [0,1]³ as a placeable definition.
func unitBoxDef(t *testing.T) fakeBodyDef {
	t.Helper()
	bodies := topo.NewSurfaceBodies()
	bodies.Add(unitBlock(t))
	return fakeBodyDef{bodies: bodies}
}

// TestAssemblyProxyCutResolvesProxiedTool gates the proxy-input cut against the analytic
// value: a source unit box placed straddling the top half of a participant removes 0.5;
// moving the source clear of the participant (re-resolved through the proxy context)
// removes nothing — the cut is associative.
func TestAssemblyProxyCutResolvesProxiedTool(t *testing.T) {
	t.Parallel()
	occs := occurrence.NewOccurrences()
	src := occs.AddByComponentDefinition("src:1", unitBoxDef(t), math.Translation4(math.V3(0, 0, 0.5)))
	f := NewAssemblyProxyCutFeature(src, ops.Cut)

	out, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 || stdmath.Abs(bodyVolume(out.Bodies[0])-0.5) > 1e-6 {
		t.Fatalf("proxied cut volume = %v, want one body of 0.5", volumesOf(out.Bodies))
	}

	src.SetTransform(math.Translation4(math.V3(0, 0, 5))) // move the source clear
	out2, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute after move: %v", err)
	}
	if len(out2.Bodies) != 1 || stdmath.Abs(bodyVolume(out2.Bodies[0])-1.0) > 1e-6 {
		t.Errorf("after moving source clear: volume = %v, want 1.0 (associative, no overlap)", volumesOf(out2.Bodies))
	}
}

// TestAssemblyProxyCutMissingBodiesFails: a source definition with no bodies is a lost
// input reported as an error, not a panic.
func TestAssemblyProxyCutMissingBodiesFails(t *testing.T) {
	t.Parallel()
	occs := occurrence.NewOccurrences()
	src := occs.AddByComponentDefinition("empty:1", boxOnlyDef{}, math.Identity4())
	f := NewAssemblyProxyCutFeature(src, ops.Cut)
	if _, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}}); err == nil {
		t.Fatal("Recompute with a bodiless source returned nil error, want a failure")
	}
}

// boxOnlyDef is a definition that has a range box but no bodies (not an occurrenceBodies).
type boxOnlyDef struct{}

func (boxOnlyDef) RangeBox() math.Box { return math.EmptyBox() }

// volumesOf lists each body's volume (for failure messages).
func volumesOf(bodies []*topo.Body) []float64 {
	out := make([]float64, len(bodies))
	for i, b := range bodies {
		out[i] = bodyVolume(b)
	}
	return out
}
