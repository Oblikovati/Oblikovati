// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// #1521 (loft UI re-flow): the Sections list needs a per-row API — a label and kind to render each
// row, a per-row delete, and drag-to-reorder. These lock that API; the head dialog draws on top of it.

// loftWithThreeProfiles builds a loft tool holding three stacked square profiles (z = 0, 5, 10), each
// on its own named sketch, returning the tool and the three sketch names in pick order.
func loftWithThreeProfiles(t *testing.T) (*LoftTool, []string) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	zs := []float64{0, 5, 10}
	l := NewLoftTool()
	names := make([]string, len(zs))
	for i, z := range zs {
		sk := centeredSquareSketch(def, planeAtZ(z), 2)
		names[i] = sk.Name()
		l.Pick(s, ProfileHandle{Sketch: sk, ProfileIndex: 0})
	}
	return l, names
}

// TestLoftSectionLabelsAreSourceNames checks a profile section is labelled with its source sketch name
// (Inventor lists the contributing sketch), in pick order.
func TestLoftSectionLabelsAreSourceNames(t *testing.T) {
	l, names := loftWithThreeProfiles(t)
	if l.SectionCount() != 3 {
		t.Fatalf("section count = %d, want 3", l.SectionCount())
	}
	for i, want := range names {
		if got := l.SectionLabel(i); got != want {
			t.Errorf("section %d label = %q, want %q (the source sketch name)", i, got, want)
		}
		if k := l.SectionKindAt(i); k != LoftSectionProfile {
			t.Errorf("section %d kind = %v, want LoftSectionProfile", i, k)
		}
	}
	if l.SectionLabel(99) != "" || l.SectionLabel(-1) != "" {
		t.Error("out-of-range SectionLabel should be empty")
	}
}

// TestLoftRemoveSection deletes the middle section and checks the rest keep order.
func TestLoftRemoveSection(t *testing.T) {
	l, names := loftWithThreeProfiles(t)
	l.RemoveSection(1) // drop the middle
	if l.SectionCount() != 2 {
		t.Fatalf("after remove, count = %d, want 2", l.SectionCount())
	}
	if l.SectionLabel(0) != names[0] || l.SectionLabel(1) != names[2] {
		t.Errorf("after removing the middle, sections = [%q %q], want [%q %q]",
			l.SectionLabel(0), l.SectionLabel(1), names[0], names[2])
	}
	before := l.SectionCount()
	l.RemoveSection(99) // out of range
	l.RemoveSection(-1)
	if l.SectionCount() != before {
		t.Error("out-of-range RemoveSection must be a no-op")
	}
}

// TestLoftMoveSection reorders the list (the blend order) and checks the new sequence.
func TestLoftMoveSection(t *testing.T) {
	l, names := loftWithThreeProfiles(t)
	l.MoveSection(2, 0) // move the last section to the front: [c a b]
	want := []string{names[2], names[0], names[1]}
	for i, w := range want {
		if got := l.SectionLabel(i); got != w {
			t.Errorf("after Move(2,0), section %d = %q, want %q", i, got, w)
		}
	}
	order := loftLabels(l)
	l.MoveSection(0, 0) // no-op
	l.MoveSection(5, 0) // out of range
	if got := loftLabels(l); !equalStrings(got, order) {
		t.Errorf("degenerate MoveSection changed order: %v, want %v", got, order)
	}
}

func loftLabels(l *LoftTool) []string {
	out := make([]string, l.SectionCount())
	for i := range out {
		out[i] = l.SectionLabel(i)
	}
	return out
}

// TestLoftTransitionMapping locks the Transition-tab map-curve API: a fresh loft maps automatically
// (no map curves); arming routes path picks to map curves; clearing returns to automatic (#1521).
func TestLoftTransitionMapping(t *testing.T) {
	l := NewLoftTool()
	if !l.AutomaticMapping() {
		t.Fatal("a fresh loft should map automatically (no map curves)")
	}
	l.ArmMapCurvePicking()
	if l.GuideKind() != loftGuideMapCurve {
		t.Errorf("ArmMapCurvePicking routed to kind %d, want map-curve (%d)", l.GuideKind(), loftGuideMapCurve)
	}
	// a picked open path becomes a map curve
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, _ := s.Workspace().Add(doc.Part, "part.obk", true)
	pd.SetContent(def)
	ps := def.Sketches().Add(sketchPlaneForMapCurve())
	a := ps.Points().Add(mapCurveP2(0, 0))
	b := ps.Points().Add(mapCurveP2(0, 5))
	ps.Lines().Add(a, b)
	l.Pick(s, PathHandle{Sketch: ps, PathIndex: 0})
	if l.AutomaticMapping() || l.MapCurveCount() != 1 {
		t.Errorf("after picking a map curve: automatic=%v count=%d, want (false, 1)", l.AutomaticMapping(), l.MapCurveCount())
	}
	l.ClearMapCurves()
	if !l.AutomaticMapping() {
		t.Error("ClearMapCurves should return the loft to automatic mapping")
	}
}

func sketchPlaneForMapCurve() sketch.Plane { return sketch.XZPlane() }
func mapCurveP2(x, y float64) math.Point2  { return math.P2(math.Scalar(x), math.Scalar(y)) }
