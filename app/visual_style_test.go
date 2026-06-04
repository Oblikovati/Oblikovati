// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/renderer"
)

// TestVisualStyleDefault checks a new session draws Shaded with Edges (so model edges are
// visible by default).
func TestVisualStyleDefault(t *testing.T) {
	if s := NewSession(); s.VisualStyle() != renderer.ShadedWithEdges {
		t.Errorf("default visual style = %v, want Shaded with Edges", s.VisualStyle())
	}
}

// TestVisualStyleCommands checks the View-tab Visual Style commands set the session style.
func TestVisualStyleCommands(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	cases := map[string]renderer.VisualStyle{
		"View.Shaded":          renderer.Shaded,
		"View.Wireframe":       renderer.Wireframe,
		"View.ShadedWithEdges": renderer.ShadedWithEdges,
		"View.Realistic":       renderer.Realistic,
		"View.Monochrome":      renderer.Monochrome,
		"View.Watercolor":      renderer.Watercolor,
	}
	for cmd, want := range cases {
		if err := s.Execute(cmd); err != nil {
			t.Fatalf("execute %s: %v", cmd, err)
		}
		if s.VisualStyle() != want {
			t.Errorf("after %s, visual style = %v, want %v", cmd, s.VisualStyle(), want)
		}
	}
}

// visualStylePanel finds the View tab's Visual Style panel in a freshly built ribbon.
func visualStylePanel(t *testing.T, s *Session) RibbonPanel {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab("View")
	if !ok {
		t.Fatal("no View tab on the ribbon")
	}
	p, ok := tab.Panel("Visual Style")
	if !ok {
		t.Fatal("no Visual Style panel on the View tab")
	}
	return p
}

// TestVisualStyleIsSelectionBox checks the Visual Style panel renders as a selection box (a
// RibbonSelector) listing every display mode — not a button grid — and that its current
// selection tracks the session's visual style.
func TestVisualStyleIsSelectionBox(t *testing.T) {
	s := registeredSession(t) // a part session, so the Part ribbon's View tab is built
	p := visualStylePanel(t, s)
	if p.Selector == nil {
		t.Fatal("Visual Style panel is not a selection box (Selector is nil)")
	}
	if len(p.Selector.Options) != 11 {
		t.Errorf("selection box has %d options, want 11 (full DisplayModeEnum)", len(p.Selector.Options))
	}
	// Default style is Shaded with Edges → the box previews it.
	if got := p.Selector.Options[p.Selector.SelectedIndex].Label; got != renderer.ShadedWithEdges.String() {
		t.Errorf("default selection = %q, want %q", got, renderer.ShadedWithEdges.String())
	}
	// Selecting Wireframe moves the highlighted option.
	if err := s.Execute("View.Wireframe"); err != nil {
		t.Fatalf("execute View.Wireframe: %v", err)
	}
	p = visualStylePanel(t, s)
	if got := p.Selector.Options[p.Selector.SelectedIndex].Label; got != renderer.Wireframe.String() {
		t.Errorf("after selecting Wireframe, selection = %q, want %q", got, renderer.Wireframe.String())
	}
}
