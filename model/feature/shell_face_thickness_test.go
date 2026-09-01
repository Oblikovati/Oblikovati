// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// A shell's per-face wall thicknesses at the FEATURE level (#1864): driven by parameters like
// every other dimension, and carried through the recipe.

// shellSeed returns a part holding a 4×4×4 box, with the box for picking face keys.
func shellSeed(t *testing.T) (*PartFeatures, *topo.Body) {
	t.Helper()
	box := subd.ToBody(subd.Box(4, 4, 4), "box")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	return fs, box
}

// boxFaceKey returns the key of the box face whose outward normal points along (nx,ny,nz).
func boxFaceKey(t *testing.T, b *topo.Body, nx, ny, nz float64) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		n := f.Geometry().NormalAt(0, 0)
		if float64(n.X)*nx+float64(n.Y)*ny+float64(n.Z)*nz > 0.99 {
			return f.ReferenceKey()
		}
	}
	t.Fatalf("no face with normal (%g,%g,%g)", nx, ny, nz)
	return nil
}

// TestShellFaceThicknessIsParameterDriven: the override is a closure, so re-evaluating the
// feature has to re-read it. A value captured once at authoring time would leave the thick wall
// behind when the parameter that sizes it changes.
func TestShellFaceThicknessIsParameterDriven(t *testing.T) {
	t.Parallel()
	fs, box := shellSeed(t)
	wall := 1.5
	sh := NewDressUpFeatures(fs).AddShell([][]byte{boxFaceKey(t, box, 0, 0, 1)}, constFloat(0.5))
	sh.Definition().(*ShellFeature).Definition().FaceThicknesses = []ShellFaceThickness{
		{FaceKey: boxFaceKey(t, box, 1, 0, 0), Thickness: func() float64 { return wall }},
	}
	fs.Recompute()
	if !sh.Health().OK() {
		t.Fatalf("shell with a face thickness went sick: %+v", sh.Health())
	}
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-43) > 1e-6 {
		t.Fatalf("shell volume = %g, want 43 (a 1.5 wall on +X)", got)
	}
	wall = 0.25      // the parameter moves
	fs.MarkDirty(sh) // as a parameter change does
	fs.Recompute()
	if got := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(got-29.875) > 1e-6 {
		t.Errorf("after the parameter changed, volume = %g, want 29.875 — the override was frozen", got)
	}
}

// TestShellFaceThicknessRoundTrips: a reopened document must shell the same walls. Losing the
// overrides would silently rebuild the part at the uniform thickness.
func TestShellFaceThicknessRoundTrips(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewDressUpFeatures(fs).addShell(&ShellDefinition{
		RemovedFaceKeys: [][]byte{[]byte("top")}, Thickness: constFloat(0.5),
		Direction: ops.ShellOutside,
		FaceThicknesses: []ShellFaceThickness{
			{FaceKey: []byte("side"), Thickness: constFloat(1.5)},
		},
	})
	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	if d := data[0].Shell.FaceThicknesses; len(d) != 1 || d[0].Thickness != 1.5 {
		t.Fatalf("serialized face thicknesses = %+v, want one 1.5 entry", d)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*ShellFeature).Definition()
	if len(def.FaceThicknesses) != 1 || string(def.FaceThicknesses[0].FaceKey) != "side" {
		t.Fatalf("restored face thicknesses = %+v, want the side override", def.FaceThicknesses)
	}
	if got := def.FaceThicknesses[0].Thickness(); got != 1.5 {
		t.Errorf("restored override thickness = %g, want 1.5", got)
	}
}
