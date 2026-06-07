// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"oblikovati/script"
)

// globalsWithMethods builds Globals exposing both the fake call door and a fixed method
// list, so the typed group sugar (oblikovati.<group>.<method>) is installed.
func globalsWithMethods(f *fakeCall, methods ...string) script.Globals {
	return script.Globals{
		Call:    f.call,
		Print:   func(string) {},
		Methods: func() []string { return methods },
	}
}

// TestTypedGroupForwardsToCall: oblikovati.documents.create{…} forwards to the host with
// the fixed method name and the args table, and returns the decoded reply — identical to
// oblikovati.call("documents.create", {…}) but ergonomic.
func TestTypedGroupForwardsToCall(t *testing.T) {
	f := &fakeCall{reply: []byte(`{"id":9}`)}
	g := globalsWithMethods(f, "documents.create", "documents.list")
	src := `local r = oblikovati.documents.create{ type = "part", name = "lid" }
	        print(r.id)`
	res := New().Run(context.Background(), src, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("Run: %v", res.Err)
	}
	if f.method != "documents.create" {
		t.Errorf("method = %q, want documents.create", f.method)
	}
	var got map[string]any
	if err := json.Unmarshal(f.argsJSON, &got); err != nil {
		t.Fatalf("args not JSON: %v", err)
	}
	if got["type"] != "part" || got["name"] != "lid" {
		t.Errorf("args round-trip wrong: %v", got)
	}
	if !strings.Contains(res.Stdout, "9") {
		t.Errorf("typed group did not return the decoded reply: %q", res.Stdout)
	}
}

// TestTypedGroupNoArgs: a method invoked with no argument table sends an empty object, so
// argument-less calls like oblikovati.documents.list() work.
func TestTypedGroupNoArgs(t *testing.T) {
	f := &fakeCall{reply: []byte(`{"documents":[]}`)}
	g := globalsWithMethods(f, "documents.list")
	res := New().Run(context.Background(), `oblikovati.documents.list()`, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("Run: %v", res.Err)
	}
	if f.method != "documents.list" {
		t.Errorf("method = %q, want documents.list", f.method)
	}
	if strings.TrimSpace(string(f.argsJSON)) != "{}" {
		t.Errorf("no-arg call should send an empty object, got %q", f.argsJSON)
	}
}

// TestTypedGroupsShareNamespace: two methods in the same group land under one table.
func TestTypedGroupsShareNamespace(t *testing.T) {
	f := &fakeCall{reply: []byte(`{}`)}
	g := globalsWithMethods(f, "sketch.rectangle", "sketch.line")
	src := `assert(type(oblikovati.sketch) == "table")
	        assert(type(oblikovati.sketch.rectangle) == "function")
	        assert(type(oblikovati.sketch.line) == "function")`
	res := New().Run(context.Background(), src, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("both methods should share the sketch table: %v", res.Err)
	}
}

// TestTypedGroupErrorIsCatchable: a host error from a typed-group call is raised as a Lua
// error the script can pcall, same as the generic door.
func TestTypedGroupErrorIsCatchable(t *testing.T) {
	f := &fakeCall{err: errBoom}
	g := globalsWithMethods(f, "documents.create")
	src := `local ok = pcall(function() oblikovati.documents.create{} end)
	        if ok then error("expected failure") end
	        print("caught")`
	res := New().Run(context.Background(), src, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("pcall should let the script handle the host error: %v", res.Err)
	}
	if !strings.Contains(res.Stdout, "caught") {
		t.Errorf("typed-group host error not catchable: %q", res.Stdout)
	}
}

// TestCallAndMethodsSurviveTypedGroups: installing the groups must not shadow the reserved
// oblikovati.call / oblikovati.methods entries.
func TestCallAndMethodsSurviveTypedGroups(t *testing.T) {
	f := &fakeCall{reply: []byte(`{}`)}
	g := globalsWithMethods(f, "documents.list")
	src := `assert(type(oblikovati.call) == "function")
	        assert(type(oblikovati.methods) == "function")
	        assert(#oblikovati.methods() == 1)`
	res := New().Run(context.Background(), src, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("reserved entries should survive: %v", res.Err)
	}
}

// TestNoTypedGroupsWithoutMethods: with no Methods provider, no group tables exist (only the
// generic call door), so the sugar is opt-in on discoverability being wired.
func TestNoTypedGroupsWithoutMethods(t *testing.T) {
	f := &fakeCall{reply: []byte(`{}`)}
	g := script.Globals{Call: f.call, Print: func(string) {}}
	res := New().Run(context.Background(), `assert(oblikovati.documents == nil)`, g, script.Limits{Wall: time.Second})
	if res.Err != nil {
		t.Fatalf("no groups should exist without a Methods provider: %v", res.Err)
	}
}
