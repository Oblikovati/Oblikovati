// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati/addin/opregistry"
	"oblikovati/api/types"
	"oblikovati/api/wire"
	"oblikovati/app"
)

// hasTab reports whether the ribbon result carries a tab of the given name.
func hasTab(r wire.ListRibbonResult, name string) bool {
	for _, t := range r.Tabs {
		if t.Name == name {
			return true
		}
	}
	return false
}

// TestRibbonListPartRibbon: with a part active, ribbon.list returns the Part ribbon and its
// modeling tabs, and not the contextual Sketch tab (no sketch open).
func TestRibbonListPartRibbon(t *testing.T) {
	r, s := seededSession(t)
	var res wire.ListRibbonResult
	call(t, r, s, "ribbon.list", "{}", &res)
	if res.Key != types.PartRibbon {
		t.Fatalf("ribbon key = %q, want Part", res.Key)
	}
	if !hasTab(res, "3D Model") {
		t.Error("Part ribbon has no 3D Model tab")
	}
	if hasTab(res, "Sketch") {
		t.Error("Sketch tab should be contextual (absent outside the sketch environment)")
	}
}

// TestRibbonListZeroDoc: with no document open, ribbon.list returns the ZeroDoc ribbon with its
// Get Started tab — the discovery surface an add-in reads before any document exists.
func TestRibbonListZeroDoc(t *testing.T) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	r := New(opregistry.Default())
	var res wire.ListRibbonResult
	call(t, r, s, "ribbon.list", "{}", &res)
	if res.Key != types.ZeroDocRibbon {
		t.Fatalf("ribbon key = %q, want ZeroDoc", res.Key)
	}
	if !hasTab(res, "Get Started") {
		t.Error("ZeroDoc ribbon has no Get Started tab")
	}
}

// TestCreateCommandTargetsRibbon: an add-in places a button on the ZeroDoc ribbon, and
// ribbon.list (with no document open) then reports it — the end-to-end "add my button to a
// named ribbon" path.
func TestCreateCommandTargetsRibbon(t *testing.T) {
	s := app.NewSession()
	r := New(opregistry.Default())
	call(t, r, s, "commands.create",
		`{"id":"acme.start","displayName":"Acme Start","ribbon":"ZeroDoc","tab":"Get Started","category":"Acme"}`, nil)
	var res wire.ListRibbonResult
	call(t, r, s, "ribbon.list", "{}", &res)
	for _, tab := range res.Tabs {
		for _, p := range tab.Panels {
			for _, ctl := range p.Controls {
				if ctl.CommandID == "acme.start" {
					return // the add-in's button landed on the ZeroDoc ribbon
				}
			}
		}
	}
	t.Fatalf("acme.start not found on the ZeroDoc ribbon: %+v", res)
}

// TestCreateCommandRejectsUnknownRibbon: a typo'd ribbon name is refused rather than silently
// dropping the button onto the default ribbon.
func TestCreateCommandRejectsUnknownRibbon(t *testing.T) {
	s := app.NewSession()
	r := New(opregistry.Default())
	if _, err := r.Handle(s, "commands.create",
		[]byte(`{"id":"x","displayName":"X","ribbon":"Bogus"}`)); err == nil {
		t.Fatal("commands.create accepted an unknown ribbon name")
	}
}

// TestCommandsListReportsRibbonAndEnvironment: commands.list surfaces the ribbon and
// environment so an add-in can see where existing commands live — a base Part command and a
// contextual Sketch command.
func TestCommandsListReportsRibbonAndEnvironment(t *testing.T) {
	r, s := seededSession(t)
	var res wire.ListCommandsResult
	call(t, r, s, "commands.list", "{}", &res)
	byID := map[string]wire.CommandInfo{}
	for _, c := range res.Commands {
		byID[c.ID] = c
	}
	extrude, ok := byID["Create.Extrude"]
	if !ok || extrude.Ribbon != types.PartRibbon || extrude.Environment != types.BaseEnvironment {
		t.Errorf("Extrude = %+v, want Part ribbon / base environment", extrude)
	}
	line, ok := byID["Sketch.Line"]
	if !ok || line.Environment != types.SketchEnvironment {
		t.Errorf("Sketch.Line = %+v, want sketch environment", line)
	}
}

// TestSetCommandState checks an add-in can toggle its command's active flag and relabel it
// (the meeting add-in's Presenter/Follow toggles) via commands.setState.
func TestSetCommandState(t *testing.T) {
	s := app.NewSession()
	r := New(opregistry.Default())
	call(t, r, s, "commands.create",
		`{"id":"acme.toggle","displayName":"Toggle","ribbon":"Part","tab":"Add-Ins","category":"Acme"}`, nil)
	call(t, r, s, "commands.setState", `{"id":"acme.toggle","active":true,"displayName":"On"}`, nil)

	cmd, ok := s.Commands().ByID("acme.toggle")
	if !ok {
		t.Fatal("acme.toggle should exist")
	}
	if !cmd.IsActive(s) {
		t.Error("setState active=true should make IsActive true")
	}
	if cmd.DisplayName() != "On" {
		t.Errorf("displayName = %q, want On", cmd.DisplayName())
	}
	call(t, r, s, "commands.setState", `{"id":"acme.toggle","active":false}`, nil)
	if cmd.IsActive(s) {
		t.Error("setState active=false should clear IsActive")
	}
	if cmd.DisplayName() != "On" {
		t.Errorf("empty displayName should keep the label, got %q", cmd.DisplayName())
	}
}
