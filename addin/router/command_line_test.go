// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// sketchEditSession builds a router + session whose part has an open XY sketch, so the
// command-line LINE flow (which needs the sketch environment) can be driven over the wire.
func sketchEditSession(t *testing.T) (*Router, *app.Session, *sketch.Sketch) {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d, err := s.Workspace().Add(doc.Part, "cmdline.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	d.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	return New(opregistry.Default()), s, sk
}

func TestSubmitCommandLineDrawsLineOverWire(t *testing.T) {
	t.Parallel()
	r, s, sk := sketchEditSession(t)

	var res wire.CommandLineResult
	call(t, r, s, "commandLine.submit", `{"line":"LINE"}`, &res)
	if !res.Awaiting || res.Prompt == "" {
		t.Fatalf("after LINE: awaiting=%v prompt=%q, want awaiting + a prompt", res.Awaiting, res.Prompt)
	}
	if len(res.Output) == 0 {
		t.Error("LINE produced no scrollback output")
	}

	// LINE is a continuous chain; an empty submit (Enter) ends it.
	call(t, r, s, "commandLine.submit", `{"line":"0,0"}`, &res)
	call(t, r, s, "commandLine.submit", `{"line":"10,0"}`, &res)
	call(t, r, s, "commandLine.submit", `{"line":""}`, &res)
	if res.Awaiting {
		t.Error("line should have committed after Enter (not awaiting)")
	}
	if sk.Lines().Count() != 1 {
		t.Errorf("got %d lines over the wire, want 1", sk.Lines().Count())
	}
}

func TestSubmitCommandLineUnknownReturnsErrorField(t *testing.T) {
	t.Parallel()
	r, s, _ := sketchEditSession(t)
	var res wire.CommandLineResult
	// An unknown command is a normal result with Error set, not a transport failure.
	call(t, r, s, "commandLine.submit", `{"line":"FLERP"}`, &res)
	if res.Error == "" {
		t.Error("unknown command should populate the result Error field")
	}
	if res.Awaiting {
		t.Error("a failed command should not leave the engine awaiting")
	}
}
