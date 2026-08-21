// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// straddleSketch is a 2D sketch on the XY plane whose rectangle [1,3]×[−1,5] straddles the box in Y,
// so projecting it +Z scores the box's top/bottom caps edge-to-edge (at x=1 and x=3) rather than
// leaving an interior island.
func straddleSketch() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	c := []*sketch.Point{
		s.Points().Add(math.P2(1, -1)),
		s.Points().Add(math.P2(3, -1)),
		s.Points().Add(math.P2(3, 5)),
		s.Points().Add(math.P2(1, 5)),
	}
	s.Lines().Add(c[0], c[1])
	s.Lines().Add(c[1], c[2])
	s.Lines().Add(c[2], c[3])
	s.Lines().Add(c[3], c[0])
	return s
}

// TestSplitFacesByPathScoresTheBox is the #2068 feature acceptance: the SplitByPath tool projects a
// sketch profile onto the running solid and scores its faces, adding faces while removing no
// material and keeping one body.
func TestSplitFacesByPathScoresTheBox(t *testing.T) {
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(boxBody()) // [0,4]³
	split := NewModifyFeatures(fs).AddSplitFacesByPath(straddleSketch(), 0)
	fs.Recompute()

	if !split.Health().OK() {
		t.Fatalf("split-by-path went sick: %s", split.Health().Reason)
	}
	pieces := fs.Result()
	if len(pieces) != 1 {
		t.Fatalf("split-by-path result = %d bodies, want 1 (no material removed)", len(pieces))
	}
	body := pieces[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("imprinted body is not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-64) > 1e-6 {
		t.Errorf("volume = %g, want 64 (a face split removes no material)", v)
	}
	if n := len(body.Faces()); n <= 6 {
		t.Errorf("scored box has %d faces, want more than the original 6", n)
	}
}

// TestSplitFacesByPathRoundTrip persists the path split and reads it back, resolving the sketch by
// index, so a document carrying a split-by-path re-resolves its profile.
func TestSplitFacesByPathRoundTrip(t *testing.T) {
	sk := straddleSketch()
	idx := oneSketch{s: sk}
	fs := NewPartFeatures(nil)
	NewModifyFeatures(fs).AddSplitFacesByPath(sk, 0)

	data, err := fs.MarshalRecipe(idx)
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, idx, NewWorkGeometry()); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	def := fresh.Item(0).Definition().(*SplitSolidFeature).Definition()
	if def.Tool != SplitByPath || def.Sketch != sk || def.ProfileIndex != 0 || !def.FacesOnly {
		t.Errorf("restored split = tool %d sketch %p profile %d facesOnly %v, want path + the sketch + 0 + true",
			def.Tool, def.Sketch, def.ProfileIndex, def.FacesOnly)
	}
}
