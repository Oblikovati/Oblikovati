// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"encoding/json"
	"fmt"
	"testing"

	"oblikovati.org/api/types"
)

// FakeManifestedAddIn is a named in-memory add-in with a manifest, a location, and
// an optional automation surface — the test double for shared-library add-ins.
type FakeManifestedAddIn struct {
	id        string
	manifest  string
	path      string
	automated bool
	calls     []string
}

func (f *FakeManifestedAddIn) ID() string                { return f.id }
func (f *FakeManifestedAddIn) Activate(*Session) error   { return nil }
func (f *FakeManifestedAddIn) Deactivate(*Session) error { return nil }
func (f *FakeManifestedAddIn) Manifest() string          { return f.manifest }
func (f *FakeManifestedAddIn) Path() string              { return f.path }
func (f *FakeManifestedAddIn) HasAutomation() bool       { return f.automated }

// CallAutomation echoes the method plus the parsed argument, so tests can assert
// the payload crossed intact.
func (f *FakeManifestedAddIn) CallAutomation(method string, args []byte) ([]byte, error) {
	f.calls = append(f.calls, method)
	return []byte(fmt.Sprintf(`{"echo":%q,"got":%s}`, method, string(args))), nil
}

// FakeBehaviorStore is a named in-memory AddInBehaviorStore.
type FakeBehaviorStore struct {
	stored map[string]types.AddInLoadBehavior
	saves  int
}

func (f *FakeBehaviorStore) Load() (map[string]types.AddInLoadBehavior, error) {
	return f.stored, nil
}

func (f *FakeBehaviorStore) Save(m map[string]types.AddInLoadBehavior) error {
	f.stored = m
	f.saves++
	return nil
}

func manifestedAddIn(id string) *FakeManifestedAddIn {
	return &FakeManifestedAddIn{
		id:   id,
		path: "/addins/" + id + ".so",
		manifest: fmt.Sprintf(
			`{"id":%q,"displayName":"Fake %s","version":"1.2.3","description":"a fake","capabilities":["commands"]}`,
			id, id),
		automated: true,
	}
}

func TestDescribeReadsManifestAndRuntimeState(t *testing.T) {
	s := NewSession()
	a := manifestedAddIn("com.x.a")
	if err := s.AddIns().Register(a); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := s.AddIns().Activate(s, "com.x.a"); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	info, err := s.AddIns().Describe("com.x.a")
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if info.DisplayName != "Fake com.x.a" || info.Version != "1.2.3" {
		t.Errorf("identity = %q/%q, want from manifest", info.DisplayName, info.Version)
	}
	if !info.Activated || !info.HasAutomation || info.Location != "/addins/com.x.a.so" {
		t.Errorf("state = %+v, want activated, hasAutomation, located", info)
	}
	if len(info.Capabilities) != 1 || info.Capabilities[0] != "commands" {
		t.Errorf("capabilities = %v, want [commands]", info.Capabilities)
	}
}

func TestDescribeUnknownAddInFails(t *testing.T) {
	if _, err := NewSession().AddIns().Describe("com.x.ghost"); err == nil {
		t.Fatal("Describe(unknown) should fail")
	}
}

func TestDescribeToleratesMalformedManifest(t *testing.T) {
	s := NewSession()
	a := &FakeManifestedAddIn{id: "com.x.bad", manifest: "{not json"}
	if err := s.AddIns().Register(a); err != nil {
		t.Fatalf("Register: %v", err)
	}
	info, err := s.AddIns().Describe("com.x.bad")
	if err != nil || info.ID != "com.x.bad" || info.DisplayName != "" {
		t.Errorf("Describe = (%+v, %v), want bare id entry without identity fields", info, err)
	}
}

func TestSetLoadBehaviorPersistsNonDefaultsOnly(t *testing.T) {
	s := NewSession()
	store := &FakeBehaviorStore{}
	if err := s.AddIns().UseBehaviorStore(store); err != nil {
		t.Fatalf("UseBehaviorStore: %v", err)
	}
	for _, id := range []string{"com.x.a", "com.x.b"} {
		if err := s.AddIns().Register(manifestedAddIn(id)); err != nil {
			t.Fatalf("Register %s: %v", id, err)
		}
	}

	if err := s.AddIns().SetLoadBehavior("com.x.a", types.LoadOnDemand); err != nil {
		t.Fatalf("SetLoadBehavior: %v", err)
	}
	if err := s.AddIns().SetLoadBehavior("com.x.b", types.LoadOnStartup); err != nil {
		t.Fatalf("SetLoadBehavior default: %v", err)
	}
	if store.saves != 2 {
		t.Errorf("saves = %d, want 2", store.saves)
	}
	if len(store.stored) != 1 || store.stored["com.x.a"] != types.LoadOnDemand {
		t.Errorf("stored = %v, want only the non-default com.x.a=demand", store.stored)
	}
	if s.AddIns().LoadBehavior("com.x.b") != types.LoadOnStartup {
		t.Error("default behavior should read back LoadOnStartup")
	}
}

func TestSetLoadBehaviorUnknownAddInFails(t *testing.T) {
	if err := NewSession().AddIns().SetLoadBehavior("com.x.ghost", types.LoadDisabled); err == nil {
		t.Fatal("SetLoadBehavior(unknown) should fail")
	}
}

func TestUseBehaviorStoreSeedsStoredBehaviors(t *testing.T) {
	s := NewSession()
	store := &FakeBehaviorStore{stored: map[string]types.AddInLoadBehavior{
		"com.x.a": types.LoadDisabled,
	}}
	if err := s.AddIns().UseBehaviorStore(store); err != nil {
		t.Fatalf("UseBehaviorStore: %v", err)
	}
	if got := s.AddIns().LoadBehavior("com.x.a"); got != types.LoadDisabled {
		t.Errorf("LoadBehavior = %v, want disabled (seeded from store)", got)
	}
}

func TestActivateRefusesDisabledAddIn(t *testing.T) {
	s := NewSession()
	if err := s.AddIns().Register(manifestedAddIn("com.x.a")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := s.AddIns().SetLoadBehavior("com.x.a", types.LoadDisabled); err != nil {
		t.Fatalf("SetLoadBehavior: %v", err)
	}
	if err := s.AddIns().Activate(s, "com.x.a"); err == nil {
		t.Fatal("Activate(disabled) should fail")
	}
	if s.AddIns().IsActive("com.x.a") {
		t.Error("disabled add-in must not be active")
	}
}

// TestCallAutomationRoutesBetweenAddIns is the #252 acceptance at the registry
// level: one add-in reaches another's automation surface through the host.
func TestCallAutomationRoutesBetweenAddIns(t *testing.T) {
	s := NewSession()
	target := manifestedAddIn("com.x.calc")
	for _, a := range []AddIn{manifestedAddIn("com.x.caller"), target} {
		if err := s.AddIns().Register(a); err != nil {
			t.Fatalf("Register: %v", err)
		}
		if err := s.AddIns().Activate(s, a.ID()); err != nil {
			t.Fatalf("Activate: %v", err)
		}
	}

	out, err := s.AddIns().CallAutomation("com.x.calc", "add", []byte(`{"a":3}`))
	if err != nil {
		t.Fatalf("CallAutomation: %v", err)
	}
	var reply struct {
		Echo string          `json:"echo"`
		Got  json.RawMessage `json:"got"`
	}
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatalf("reply not JSON: %v", err)
	}
	if reply.Echo != "add" || string(reply.Got) != `{"a":3}` {
		t.Errorf("reply = %+v, want method add with payload {\"a\":3}", reply)
	}
	if len(target.calls) != 1 {
		t.Errorf("target saw %d calls, want 1", len(target.calls))
	}
}

func TestCallAutomationRequiresActiveTarget(t *testing.T) {
	s := NewSession()
	if err := s.AddIns().Register(manifestedAddIn("com.x.calc")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, err := s.AddIns().CallAutomation("com.x.calc", "add", nil); err == nil {
		t.Fatal("CallAutomation on an inactive add-in should fail (mirrors ApplicationAddIn.Automation)")
	}
}

func TestCallAutomationHonorsProbe(t *testing.T) {
	s := NewSession()
	a := manifestedAddIn("com.x.mute")
	a.automated = false // the wrapper has the method but the export is absent
	if err := s.AddIns().Register(a); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := s.AddIns().Activate(s, "com.x.mute"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if _, err := s.AddIns().CallAutomation("com.x.mute", "x", nil); err == nil {
		t.Fatal("CallAutomation should fail when the probe reports no automation")
	}
	info, _ := s.AddIns().Describe("com.x.mute")
	if info.HasAutomation {
		t.Error("Describe should report hasAutomation=false from the probe")
	}
}
