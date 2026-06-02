// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// seededSession builds a router and a session with an active part that has a "width"
// parameter and one sketch holding a closed 4x3 rectangle profile — the same shape
// the head seeds, so feature operations have real geometry to work on.
func seededSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d, err := s.Workspace().Add(doc.Part, "test.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	d.SetContent(def)
	if _, err := def.Parameters().AddUserParameter("width", "4 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	addRect(def.Sketches().Add(sketch.XYPlane()), 4, 3)
	def.Recompute()
	return New(opregistry.Default()), s
}

// addRect draws a closed w×h rectangle at the sketch origin (one profile).
func addRect(sk *sketch.Sketch, w, h float64) {
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(w, 0))
	c2 := sk.Points().Add(math.P2(w, h))
	c3 := sk.Points().Add(math.P2(0, h))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// call invokes a method and unmarshals the JSON result into v (nil to ignore).
func call(t *testing.T, r *Router, s *app.Session, method, args string, v any) {
	t.Helper()
	out, err := r.Handle(s, method, []byte(args))
	if err != nil {
		t.Fatalf("%s(%s): %v", method, args, err)
	}
	if v != nil {
		if err := json.Unmarshal(out, v); err != nil {
			t.Fatalf("%s: unmarshal result: %v", method, err)
		}
	}
}

func TestUnknownMethod(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "bogus.method", nil); err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestCommandsListAndExecute(t *testing.T) {
	r, s := seededSession(t)
	if err := s.Commands().Add(app.NewCommand("test.noop", "Noop", "Test", func(*app.Session) error { return nil })); err != nil {
		t.Fatalf("add test command: %v", err)
	}
	var res wire.ListCommandsResult
	call(t, r, s, "commands.list", "{}", &res)
	if len(res.Commands) == 0 {
		t.Fatal("commands.list returned none")
	}
	call(t, r, s, "commands.execute", `{"id":"test.noop"}`, nil)
	if _, err := r.Handle(s, "commands.execute", []byte(`{"id":"does.not.exist"}`)); err == nil {
		t.Fatal("expected error executing unknown command")
	}
}

func TestDocumentsCreateListActivate(t *testing.T) {
	r, s := seededSession(t)
	var created wire.DocumentInfo
	call(t, r, s, "documents.create", `{"type":"part","name":"second.obk"}`, &created)
	if !created.Active || created.Type != "part" {
		t.Fatalf("created doc = %+v, want active part", created)
	}
	var list wire.ListDocumentsResult
	call(t, r, s, "documents.list", "{}", &list)
	if len(list.Documents) != 2 {
		t.Fatalf("documents.list = %d, want 2", len(list.Documents))
	}
	// Activate the currently-inactive document by its real id (ids are minted from a
	// global counter, so don't assume a value) and confirm it becomes active.
	var target uint64
	for _, d := range list.Documents {
		if !d.Active {
			target = d.ID
		}
	}
	call(t, r, s, "documents.activate", `{"id":`+strconv.FormatUint(target, 10)+`}`, nil)
	call(t, r, s, "documents.list", "{}", &list)
	for _, d := range list.Documents {
		if d.ID == target && !d.Active {
			t.Fatalf("documents.activate did not activate id %d", target)
		}
	}
}

func TestParametersAddGetSet(t *testing.T) {
	r, s := seededSession(t)
	var list wire.ListParametersResult
	call(t, r, s, "parameters.list", "{}", &list)
	if len(list.Parameters) != 1 || list.Parameters[0].Name != "width" {
		t.Fatalf("parameters.list = %+v, want [width]", list.Parameters)
	}
	var added wire.ParameterInfo
	call(t, r, s, "parameters.add", `{"name":"height","expression":"3 cm"}`, &added)
	if added.Name != "height" || added.Expression != "3 cm" {
		t.Fatalf("added = %+v, want height=3 cm", added)
	}
	var got wire.ParameterInfo
	call(t, r, s, "parameters.get", `{"name":"width"}`, &got)
	if got.Expression != "4 cm" {
		t.Fatalf("width expression = %q, want \"4 cm\"", got.Expression)
	}
	var set wire.ParameterInfo
	call(t, r, s, "parameters.set", `{"name":"width","expression":"5 cm"}`, &set)
	if set.Expression != "5 cm" {
		t.Fatalf("set width expression = %q, want \"5 cm\"", set.Expression)
	}
}

func TestFeaturesListAndAddExtrude(t *testing.T) {
	r, s := seededSession(t)
	var kinds wire.ListFeatureKindsResult
	call(t, r, s, "features.list", "{}", &kinds)
	if len(kinds.Kinds) == 0 || kinds.Kinds[0].Kind != "extrude" {
		t.Fatalf("features.list = %+v, want extrude", kinds.Kinds)
	}
	if len(kinds.Kinds[0].Schema) == 0 {
		t.Fatal("extrude descriptor has no schema")
	}

	var res struct {
		Feature string `json:"feature"`
		Kind    string `json:"kind"`
		Bodies  int    `json:"bodies"`
	}
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"5 cm"}}`, &res)
	if res.Kind != "extrude" || res.Bodies != 1 {
		t.Fatalf("extrude result = %+v, want kind=extrude bodies=1", res)
	}

	var tree wire.ModelTreeResult
	call(t, r, s, "model.tree", "{}", &tree)
	if tree.Sketches != 1 || len(tree.Features) != 1 || tree.Features[0].Kind != "extrude" || tree.Bodies != 1 {
		t.Fatalf("model.tree = %+v, want 1 sketch / 1 extrude / 1 body", tree)
	}
	if !slices.Contains(tree.Parameters, "width") {
		t.Fatalf("model.tree parameters = %v, want width", tree.Parameters)
	}
}

func TestFeaturesAddUnknownKind(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "features.add", []byte(`{"kind":"nope","args":{}}`)); err == nil {
		t.Fatal("expected error for unknown feature kind")
	}
}

func TestModelSelectionEmpty(t *testing.T) {
	r, s := seededSession(t)
	var sel wire.SelectionResult
	call(t, r, s, "model.selection", "{}", &sel)
	if sel.Count != 0 {
		t.Fatalf("selection count = %d, want 0", sel.Count)
	}
}

func TestParametersNoActiveDocument(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession() // empty workspace
	if _, err := r.Handle(s, "parameters.list", nil); err == nil || !strings.Contains(err.Error(), "no active document") {
		t.Fatalf("err = %v, want no-active-document", err)
	}
}
