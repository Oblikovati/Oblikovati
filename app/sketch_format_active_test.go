// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// The Format toggles arm a creation mode when nothing is selected. Until #2041 they carried no
// WithActive predicate, so an armed mode rendered exactly like a disarmed one — nothing on
// screen said the next line would be construction geometry.

// formatButtonActive reports the ribbon Active state of a Format-panel button on the given tab.
func formatButtonActive(t *testing.T, s *Session, tab, name string) bool {
	t.Helper()
	rt, ok := BuildRibbon(s).Tab(tab)
	if !ok {
		t.Fatalf("ribbon has no %q tab", tab)
	}
	panel, ok := rt.Panel("Format")
	if !ok {
		t.Fatalf("%q tab has no Format panel", tab)
	}
	b, ok := buttonNamed(panel, name)
	if !ok {
		t.Fatalf("%q Format panel has no %q button", tab, name)
	}
	return b.Active
}

// assertArmedRendersActive runs the toggle with nothing selected — the arm branch — and checks
// the ribbon reports Active, then disarms and checks it clears.
func assertArmedRendersActive(t *testing.T, s *Session, tab, command, name string) {
	t.Helper()
	if formatButtonActive(t, s, tab, name) {
		t.Fatalf("%s starts armed; the fixture is not in the disarmed state", command)
	}
	if err := s.Execute(command); err != nil {
		t.Fatalf("execute %s: %v", command, err)
	}
	if !formatButtonActive(t, s, tab, name) {
		t.Errorf("%s is armed but its ribbon button is not Active", command)
	}
	if err := s.Execute(command); err != nil {
		t.Fatalf("execute %s (disarm): %v", command, err)
	}
	if formatButtonActive(t, s, tab, name) {
		t.Errorf("%s is disarmed but its ribbon button is still Active", command)
	}
}

func TestSketch2DFormatArmedModesRenderActive(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ command, name string }{
		{"Sketch.Construction", "Construction"},
		{"Sketch.Centerline", "Centerline"},
		{"Sketch.CenterPoint", "Center Point"},
		{"Sketch.DrivenDimension", "Driven Dimension"},
		{"Sketch.ShowFormat", "Show Format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := registeredSession(t)
			enterSketchEnv(t, s) // the Format panel is contextual to the sketch environment
			assertArmedRendersActive(t, s, "Sketch", tc.command, tc.name)
		})
	}
}

func TestSketch3DFormatArmedModesRenderActive(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ command, name string }{
		{"Sketch3D.Construction", "Construction"},
		{"Sketch3D.DrivenDimension", "Driven Dimension"},
		{"Sketch3D.ShowFormat", "Show Format"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := registeredSession(t)
			if _, err := s.CreateSketch3D(); err != nil {
				t.Fatalf("CreateSketch3D: %v", err)
			}
			assertArmedRendersActive(t, s, tab3DSketch, tc.command, tc.name)
		})
	}
}
