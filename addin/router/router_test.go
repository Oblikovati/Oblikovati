// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
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
	// The icon + button-style metadata must survive the wire round-trip so add-ins
	// can render the host's ribbon styling.
	var extrude *wire.CommandInfo
	for i := range res.Commands {
		if res.Commands[i].ID == "Create.Extrude" {
			extrude = &res.Commands[i]
		}
	}
	if extrude == nil {
		t.Fatal("commands.list omitted Create.Extrude")
	}
	if extrude.Icon != "extrude" || extrude.ButtonStyle != types.LargeIconButton {
		t.Errorf("Extrude wire info = (%q, %s), want (\"extrude\", %s)",
			extrude.Icon, extrude.ButtonStyle, types.LargeIconButton)
	}
	call(t, r, s, "commands.execute", `{"id":"test.noop"}`, nil)
	if _, err := r.Handle(s, "commands.execute", []byte(`{"id":"does.not.exist"}`)); err == nil {
		t.Fatal("expected error executing unknown command")
	}
}

// TestCommandsCreateAddsRibbonButtonAndNotifiesOnExecute proves the add-in UI
// extension contract: commands.create registers a button that appears in the ribbon,
// and executing it (a click) fires a command.ended event — the signal forwarded to an
// add-in so it can run the button's action. A duplicate id is rejected.
func TestCommandsCreateAddsRibbonButtonAndNotifiesOnExecute(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "commands.create",
		`{"id":"AddIn.Ping","displayName":"Ping","tab":"AddInTab","category":"Demo","icon":"extrude","buttonStyle":2}`, nil)

	// The add-in's button is now on the ribbon, styled as requested.
	panel, ok := app.BuildRibbon(s).Panel("Demo")
	if !ok || len(panel.Buttons) != 1 {
		t.Fatalf("Demo panel = %+v ok=%v, want one add-in button", panel, ok)
	}
	if b := panel.Buttons[0].Command; b.DisplayName() != "Ping" || b.Icon() != "extrude" || b.ButtonStyle() != app.LargeIconButton {
		t.Errorf("button = (%q,%q,%s), want (Ping,extrude,large-icon)", b.DisplayName(), b.Icon(), b.ButtonStyle())
	}

	// Clicking it fires command.ended so a listening add-in can react.
	var ended string
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.CommandEnded) event.Outcome {
		ended = e.ID
		return event.Continue()
	})
	call(t, r, s, "commands.execute", `{"id":"AddIn.Ping"}`, nil)
	if ended != "AddIn.Ping" {
		t.Errorf("command.ended id = %q, want AddIn.Ping", ended)
	}

	// A duplicate id and a missing displayName are both rejected.
	if _, err := r.Handle(s, "commands.create", []byte(`{"id":"AddIn.Ping","displayName":"Dup"}`)); err == nil {
		t.Error("duplicate commands.create accepted")
	}
	if _, err := r.Handle(s, "commands.create", []byte(`{"id":"AddIn.NoName"}`)); err == nil {
		t.Error("commands.create without displayName accepted")
	}
}

// TestCommandsCreateInlineIconSVG checks an add-in can ship its own button glyph as inline SVG: it
// is stored on the command (so the head renders it instead of a bundled key), and an oversized
// payload is rejected.
func TestCommandsCreateInlineIconSVG(t *testing.T) {
	r, s := seededSession(t)
	svg := `<svg viewBox="0 0 24 24"><rect width="24" height="24" fill="#00ff00"/></svg>`
	call(t, r, s, "commands.create",
		`{"id":"AddIn.Glyph","displayName":"Glyph","category":"Demo","iconSvg":`+jsonString(svg)+`,"buttonStyle":2}`, nil)

	panel, ok := app.BuildRibbon(s).Panel("Demo")
	if !ok || len(panel.Buttons) != 1 {
		t.Fatalf("Demo panel = %+v ok=%v, want one add-in button", panel, ok)
	}
	if got := panel.Buttons[0].Command.InlineIconSVG(); got != svg {
		t.Errorf("InlineIconSVG = %q, want the supplied markup", got)
	}

	big := `{"id":"AddIn.Big","displayName":"Big","iconSvg":` + jsonString("<svg>"+strings.Repeat("x", 17*1024)+"</svg>") + `}`
	if _, err := r.Handle(s, "commands.create", []byte(big)); err == nil {
		t.Error("oversized iconSvg accepted, want rejected")
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

func TestThemeActiveServesEveryColor(t *testing.T) {
	r, s := seededSession(t)
	var view wire.ThemeView
	call(t, r, s, "theme.active", "{}", &view)
	if view.Name != "Dark" || view.Kind != "dark" {
		t.Fatalf("theme.active = (%q,%q), want (Dark,dark)", view.Name, view.Kind)
	}
	// An add-in reads colors by token string; every token must be present so its UI can
	// look them all up without a missing-key check.
	for _, tok := range types.AllThemeTokens() {
		hex, ok := view.Colors[string(tok)]
		if !ok || len(hex) != 9 || hex[0] != '#' {
			t.Errorf("theme.active color %q = %q, want a \"#RRGGBBAA\" value", tok, hex)
		}
	}
}

func TestThemeListFlagsActive(t *testing.T) {
	r, s := seededSession(t)
	if err := s.DuplicateTheme("Dark", "My Dark"); err != nil { // becomes active
		t.Fatalf("DuplicateTheme: %v", err)
	}
	var list wire.ListThemesResult
	call(t, r, s, "theme.list", "{}", &list)
	if len(list.Themes) != 3 { // Dark, Light, My Dark
		t.Fatalf("theme.list = %d themes, want 3", len(list.Themes))
	}
	var active string
	for _, th := range list.Themes {
		if th.Active {
			active = th.Name
		}
	}
	if active != "My Dark" {
		t.Errorf("active theme in list = %q, want \"My Dark\"", active)
	}
}

func TestAppearancesAndMaterialsListAndCreate(t *testing.T) {
	r, s := seededSession(t)
	var apprs wire.ListAppearancesResult
	call(t, r, s, "appearances.list", "{}", &apprs)
	if len(apprs.Appearances) == 0 {
		t.Fatal("appearances.list returned none")
	}
	var mats wire.ListMaterialsResult
	call(t, r, s, "materials.list", "{}", &mats)
	if len(mats.Materials) == 0 {
		t.Fatal("materials.list returned none")
	}
	// Duplicate a built-in into a project-scoped editable material.
	var made wire.MaterialInfo
	call(t, r, s, "materials.create", `{"baseId":"steel","name":"Shop Steel"}`, &made)
	if made.Source != "project" || made.Density == 0 {
		t.Fatalf("created material = %+v, want a project copy with density", made)
	}
	// Edit its appearance's albedo and read it back.
	var appr wire.AppearanceInfo
	call(t, r, s, "appearances.create", `{"baseId":"steel","name":"Shop Steel Look"}`, &appr)
	appr.Albedo = "#ff8800ff"
	var updated wire.AppearanceInfo
	call(t, r, s, "appearances.update", mustJSON(t, appr), &updated)
	if updated.Albedo != "#ff8800ff" {
		t.Errorf("updated albedo = %q, want #ff8800ff", updated.Albedo)
	}
}

func TestAssignMaterialAndPhysicalProperties(t *testing.T) {
	r, s := seededSession(t)
	// Build a solid so there is volume to weigh.
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"5 cm"}}`, nil)
	call(t, r, s, "model.assignMaterial", `{"materialId":"steel"}`, nil)

	var props struct {
		Mass    float64 `json:"mass"`
		Volume  float64 `json:"volume"`
		Density float64 `json:"density"`
	}
	call(t, r, s, "model.physicalProperties", "{}", &props)
	if props.Volume <= 0 || props.Mass <= 0 {
		t.Fatalf("physical properties = %+v, want positive volume and mass", props)
	}
	// Steel density is 7.85 g/cm³, so mass ≈ density × volume.
	if got := props.Mass / props.Volume; got < 7.8 || got > 7.9 {
		t.Errorf("mass/volume = %v, want ≈7.85 (steel density)", got)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestParametersNoActiveDocument(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession() // empty workspace
	if _, err := r.Handle(s, "parameters.list", nil); err == nil || !strings.Contains(err.Error(), "no active document") {
		t.Fatalf("err = %v, want no-active-document", err)
	}
}
