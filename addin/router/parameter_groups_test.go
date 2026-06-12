// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"slices"
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
)

// Custom parameter groups over the wire (M02-F05, Oblikovati#604).

func TestParameterGroupsLifecycleOverWire(t *testing.T) {
	r, s := seededSession(t)

	var g wire.ParameterGroupInfo
	call(t, r, s, "parameters.groups.add",
		`{"internalName":"com.example:frame","displayName":"Frame","clientId":"com.example"}`, &g)
	if g.InternalName != "com.example:frame" || g.DisplayName != "Frame" || g.ClientID != "com.example" {
		t.Fatalf("added group = %+v, want key/display/client kept", g)
	}
	if _, err := r.Handle(s, "parameters.groups.add", []byte(`{"internalName":"com.example:frame"}`)); err == nil {
		t.Error("duplicate internal name must be rejected")
	}

	call(t, r, s, "parameters.groups.addMember", `{"internalName":"com.example:frame","parameter":"width"}`, &g)
	if !slices.Equal(g.Members, []string{"width"}) {
		t.Errorf("members = %v, want [width]", g.Members)
	}

	// A parameter may sit in several groups at once.
	call(t, r, s, "parameters.groups.add", `{"internalName":"aux"}`, nil)
	call(t, r, s, "parameters.groups.addMember", `{"internalName":"aux","parameter":"width"}`, nil)
	var list wire.ListParameterGroupsResult
	call(t, r, s, "parameters.groups.list", "{}", &list)
	if len(list.Groups) != 2 || len(list.Groups[0].Members) != 1 || len(list.Groups[1].Members) != 1 {
		t.Fatalf("groups = %+v, want width in both", list.Groups)
	}
	// An empty display name defaults to the internal name.
	if list.Groups[1].DisplayName != "aux" {
		t.Errorf("display name = %q, want the internal-name default", list.Groups[1].DisplayName)
	}

	// Leaving one group touches neither the parameter nor the other membership.
	var aux wire.ParameterGroupInfo
	call(t, r, s, "parameters.groups.removeMember", `{"internalName":"aux","parameter":"width"}`, &aux)
	if len(aux.Members) != 0 {
		t.Errorf("aux members after remove = %v, want none", aux.Members)
	}
	if d := getDetail(t, r, s, "width"); d.Name != "width" {
		t.Error("removing a membership must not delete the parameter")
	}
	call(t, r, s, "parameters.groups.list", "{}", &list)
	if !slices.Equal(list.Groups[0].Members, []string{"width"}) {
		t.Errorf("frame members = %v, want width kept", list.Groups[0].Members)
	}
}

func TestParameterGroupDeleteCascadeIsOptIn(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "parameters.add", `{"name":"tmp","expression":"1 cm"}`, nil)
	call(t, r, s, "parameters.groups.add", `{"internalName":"G"}`, nil)
	call(t, r, s, "parameters.groups.addMember", `{"internalName":"G","parameter":"tmp"}`, nil)

	// Plain delete: the group goes, the member stays.
	call(t, r, s, "parameters.groups.delete", `{"internalName":"G"}`, nil)
	if d := getDetail(t, r, s, "tmp"); d.Name != "tmp" {
		t.Fatal("plain group delete must keep the member parameters")
	}

	// Opt-in cascade: the member parameters go with the group.
	call(t, r, s, "parameters.groups.add", `{"internalName":"G2"}`, nil)
	call(t, r, s, "parameters.groups.addMember", `{"internalName":"G2","parameter":"tmp"}`, nil)
	call(t, r, s, "parameters.groups.delete", `{"internalName":"G2","deleteParameters":true}`, nil)
	if _, err := r.Handle(s, "parameters.getDetail", []byte(`{"name":"tmp"}`)); err == nil {
		t.Error("cascade delete must delete the member parameters")
	}
}

func TestParameterGroupSetDisplayName(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "parameters.groups.add", `{"internalName":"ratios","displayName":"Ratios"}`, nil)

	var g wire.ParameterGroupInfo
	call(t, r, s, "parameters.groups.setDisplayName", `{"internalName":"ratios","displayName":"Gear Ratios"}`, &g)
	if g.InternalName != "ratios" || g.DisplayName != "Gear Ratios" {
		t.Errorf("renamed group = %+v, want the key unchanged and the display edited", g)
	}
	if _, err := r.Handle(s, "parameters.groups.setDisplayName", []byte(`{"internalName":"ratios","displayName":""}`)); err == nil {
		t.Error("empty display name must be rejected")
	}
	if _, err := r.Handle(s, "parameters.groups.setDisplayName", []byte(`{"internalName":"nope","displayName":"x"}`)); err == nil || !strings.Contains(err.Error(), "no parameter group") {
		t.Errorf("unknown group err = %v, want a no-parameter-group rejection", err)
	}
}

func TestParameterGroupMemberRejectsUnknownNames(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "parameters.groups.add", `{"internalName":"G"}`, nil)
	for args, want := range map[string]string{
		`{"internalName":"nope","parameter":"width"}`: "no parameter group",
		`{"internalName":"G","parameter":"nope"}`:     "no parameter named",
	} {
		if _, err := r.Handle(s, "parameters.groups.addMember", []byte(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("addMember(%s) err = %v, want it to mention %q", args, err, want)
		}
	}
}

// TestParameterGroupMutationsBroadcast checks every group mutation emits
// edit.committed (the replication seam) and the list read does not.
func TestParameterGroupMutationsBroadcast(t *testing.T) {
	r, s := seededSession(t)
	var methods []string
	sub := event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
		methods = append(methods, e.Method)
		return event.Continue()
	})
	defer sub.Cancel()

	call(t, r, s, "parameters.groups.add", `{"internalName":"G"}`, nil)
	call(t, r, s, "parameters.groups.list", "{}", nil)
	call(t, r, s, "parameters.groups.setDisplayName", `{"internalName":"G","displayName":"Group"}`, nil)
	call(t, r, s, "parameters.groups.addMember", `{"internalName":"G","parameter":"width"}`, nil)
	call(t, r, s, "parameters.groups.removeMember", `{"internalName":"G","parameter":"width"}`, nil)
	call(t, r, s, "parameters.groups.delete", `{"internalName":"G"}`, nil)

	want := []string{
		wire.MethodParametersGroupsAdd, wire.MethodParametersGroupsSetDisplayName,
		wire.MethodParametersGroupsAddMember, wire.MethodParametersGroupsRemoveMember,
		wire.MethodParametersGroupsDelete,
	}
	if !slices.Equal(methods, want) {
		t.Errorf("edit.committed methods = %v, want %v (and none for list)", methods, want)
	}
}
