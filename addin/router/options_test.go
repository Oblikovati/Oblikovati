// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

func TestOptionsListGroups(t *testing.T) {
	r, s := seededSession(t)
	var res wire.ListOptionGroupsResult
	call(t, r, s, "options.listGroups", "{}", &res)
	if len(res.Groups) != 5 || res.Groups[0] != "general" || res.Groups[4] != "save" {
		t.Fatalf("groups = %v, want general/display/sketch/part/save", res.Groups)
	}
}

func TestOptionsGeneralRoundTripOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "options.setGroup", `{"group":"general","general":{"startupAction":1}}`, nil)
	var view wire.OptionGroupView
	call(t, r, s, "options.getGroup", `{"group":"general"}`, &view)
	if view.General == nil || view.General.StartupAction != types.StartupEmptyWorkspace {
		t.Fatalf("general = %+v, want startupAction=empty", view.General)
	}
}

func TestOptionsSketchAppliesToLiveGrid(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "options.setGroup",
		`{"group":"sketch","sketch":{"gridSpacingCm":2,"gridVisible":false,"gridMajorEvery":8,"snapToPoints":true,"snapToGrid":false}}`, nil)
	if s.Grid().SpacingModel() != 2 || s.Grid().Visible || s.Grid().MajorEvery != 8 {
		t.Fatalf("live grid = spacing %v visible %v major %d, want the set values applied",
			s.Grid().SpacingModel(), s.Grid().Visible, s.Grid().MajorEvery)
	}
	if _, err := r.Handle(s, "options.setGroup",
		[]byte(`{"group":"sketch","sketch":{"gridSpacingCm":-1}}`)); err == nil {
		t.Error("a non-positive grid spacing should fail over the wire")
	}
}

func TestOptionsDisplayProxiesThemeAndViewCube(t *testing.T) {
	r, s := seededSession(t)
	var before wire.OptionGroupView
	call(t, r, s, "options.getGroup", `{"group":"display"}`, &before)
	if before.Display == nil || before.Display.ColorScheme == "" {
		t.Fatalf("display = %+v, want the active theme name", before.Display)
	}

	set := *before.Display
	set.ViewCubeHidden = true
	set.CubeCorner = 3
	call(t, r, s, "options.setGroup", `{"group":"display","display":{"colorScheme":"`+set.ColorScheme+`","viewCubeHidden":true,"cubeCorner":3}}`, nil)
	if !s.ViewCubePrefs().CubeHidden || s.ViewCubePrefs().CubeCorner != 3 {
		t.Errorf("prefs = %+v, want cube hidden in corner 3", s.ViewCubePrefs())
	}
	if _, err := r.Handle(s, "options.setGroup",
		[]byte(`{"group":"display","display":{"colorScheme":"no-such-theme"}}`)); err == nil {
		t.Error("an unknown color scheme should fail")
	}
}

func TestOptionsRejectsMismatchedPayload(t *testing.T) {
	r, s := seededSession(t)
	if _, err := r.Handle(s, "options.setGroup",
		[]byte(`{"group":"general","part":{"chamferFlatCorners":false}}`)); err == nil {
		t.Error("a group naming general but carrying part should fail")
	}
	if _, err := r.Handle(s, "options.getGroup", []byte(`{"group":"bogus"}`)); err == nil {
		t.Error("an unknown group should fail")
	}
}
