//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// styledBoxSession builds a session with an extruded box and the box body selected — the state
// the Color Styles panel renders its Apply rows against.
func styledBoxSession(t *testing.T) *app.Session {
	t.Helper()
	s := framedSession()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -2))
	c1 := sk.Points().Add(math.P2(2, -2))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(-2, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	if b := s.VisibleBodies(); len(b) > 0 {
		s.Selection().Add(app.BodyHandle{Body: b[0]})
	}
	return s
}

// TestInWindowColorStylesPanelDraws opens the real window with the Color Styles panel visible
// and a selected styled body, then runs frames — exercising the style-list / Apply / Clear
// widgets without tripping Dear ImGui's Begin/End assertions (M16-F02 #403/#408).
func TestInWindowColorStylesPanelDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := styledBoxSession(t)

	key, ok := s.SelectedBodyKey()
	if !ok {
		t.Fatal("expected the box body selected")
	}
	if err := s.AssignColorStyleToBody(key, "Brass"); err != nil {
		t.Fatalf("AssignColorStyleToBody: %v", err)
	}
	s.OpenColorStylesPanel()
	defer s.CloseColorStylesPanel()

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if name, _ := s.BodyColorStyle(key); name != "Brass" {
		t.Errorf("assigned style = %q, want Brass", name)
	}
}
