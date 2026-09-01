// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"testing"

	"oblikovati.org/event"
)

func TestCommandRegistrationAndLookup(t *testing.T) {
	t.Parallel()
	m := NewCommandManager()
	line := NewCommand("Sketch.Line", "Line", "Sketch", func(*Session) error { return nil }).WithAlias("L")
	circ := NewCommand("Sketch.Circle", "Circle", "Sketch", func(*Session) error { return nil }).WithAlias("C")
	ext := NewCommand("Part.Extrude", "Extrude", "Create", func(*Session) error { return nil }).WithAlias("E")
	for _, c := range []*CommandDefinition{line, circ, ext} {
		if err := m.Add(c); err != nil {
			t.Fatalf("Add %s: %v", c.ID(), err)
		}
	}
	if got, ok := m.ByID("Part.Extrude"); !ok || got != ext {
		t.Error("ByID failed")
	}
	if got, ok := m.ByAlias("L"); !ok || got != line {
		t.Error("ByAlias failed")
	}
	if cats := m.Categories(); len(cats) != 2 || cats[0] != "Sketch" || cats[1] != "Create" {
		t.Errorf("categories = %v, want [Sketch Create]", cats)
	}
	if sk := m.ByCategory("Sketch"); len(sk) != 2 {
		t.Errorf("Sketch category has %d commands, want 2", len(sk))
	}
}

// TestCommandInlineIconSVG checks an add-in's own glyph is carried on the command: WithInlineIconSVG
// stores the markup and InlineIconSVG reads it back, distinct from the bundled-key Icon.
func TestCommandInlineIconSVG(t *testing.T) {
	t.Parallel()
	svg := `<svg viewBox="0 0 24 24"><rect width="24" height="24"/></svg>`
	c := NewCommand("AddIn.Glyph", "Glyph", "Demo", func(*Session) error { return nil }).
		WithIcon("fallback-key").WithInlineIconSVG(svg)
	if got := c.InlineIconSVG(); got != svg {
		t.Errorf("InlineIconSVG = %q, want the supplied markup", got)
	}
	if got := c.Icon(); got != "fallback-key" {
		t.Errorf("Icon = %q, want the bundled key kept alongside the inline SVG", got)
	}
	if NewCommand("x", "X", "Demo", func(*Session) error { return nil }).InlineIconSVG() != "" {
		t.Error("a command with no inline glyph should report an empty InlineIconSVG")
	}
}

func TestCommandManagerRejectsDuplicates(t *testing.T) {
	t.Parallel()
	m := NewCommandManager()
	_ = m.Add(NewCommand("x", "X", "C", func(*Session) error { return nil }).WithAlias("A"))
	if err := m.Add(NewCommand("x", "X2", "C", func(*Session) error { return nil })); err == nil {
		t.Error("duplicate id accepted")
	}
	if err := m.Add(NewCommand("y", "Y", "C", func(*Session) error { return nil }).WithAlias("A")); err == nil {
		t.Error("duplicate alias accepted")
	}
}

func TestSessionExecuteAndAliasAndEvents(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := 0
	_ = s.Commands().Add(NewCommand("greet", "Greet", "Test", func(*Session) error { ran++; return nil }).WithAlias("G"))

	var started, ended string
	var failed bool
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e CommandStarted) event.Outcome { started = e.ID; return event.Continue() })
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e CommandEnded) event.Outcome {
		ended = e.ID
		failed = e.Failed
		return event.Continue()
	})

	if err := s.Execute("greet"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if ran != 1 || started != "greet" || ended != "greet" || failed {
		t.Errorf("execute/events wrong: ran=%d started=%q ended=%q failed=%v", ran, started, ended, failed)
	}
	// Alias-driven (typed command alias).
	if err := s.Invoke("G"); err != nil || ran != 2 {
		t.Errorf("Invoke(G): err=%v ran=%d", err, ran)
	}
	// Unknown command / alias error.
	if s.Execute("nope") == nil || s.Invoke("Z") == nil {
		t.Error("unknown command/alias should error")
	}
}

func TestDisabledCommandRefuses(t *testing.T) {
	t.Parallel()
	s := NewSession()
	on := false
	ran := false
	_ = s.Commands().Add(NewCommand("c", "C", "T", func(*Session) error { ran = true; return nil }).
		WithEnable(func(*Session) bool { return on }))
	if err := s.Execute("c"); err == nil || ran {
		t.Error("disabled command ran")
	}
	on = true
	if err := s.Execute("c"); err != nil || !ran {
		t.Errorf("enabled command did not run: %v", err)
	}
}

func TestCommandEndedReportsFailure(t *testing.T) {
	t.Parallel()
	s := NewSession()
	var failed bool
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e CommandEnded) event.Outcome { failed = e.Failed; return event.Continue() })
	_ = s.Commands().Add(NewCommand("boom", "Boom", "T", func(*Session) error { return errors.New("x") }))
	if err := s.Execute("boom"); err == nil || !failed {
		t.Errorf("failing command: err=%v failed-event=%v", err, failed)
	}
}
