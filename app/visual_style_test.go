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
