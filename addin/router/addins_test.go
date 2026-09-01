// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// FakeWireAddIn is a named in-memory add-in for wire-level registry tests: it
// carries a manifest and an automation surface that doubles its numeric input.
type FakeWireAddIn struct {
	id     string
	active bool
}

func (f *FakeWireAddIn) ID() string                  { return f.id }
func (f *FakeWireAddIn) Activate(*app.Session) error { f.active = true; return nil }
func (f *FakeWireAddIn) Deactivate(*app.Session) error {
	f.active = false
	return nil
}
func (f *FakeWireAddIn) Manifest() string {
	return fmt.Sprintf(`{"id":%q,"displayName":"Wire Fake","version":"0.1.0"}`, f.id)
}
func (f *FakeWireAddIn) CallAutomation(method string, args []byte) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"method":%q,"args":%s}`, method, string(args))), nil
}

// addInSession is a session with two fake add-ins registered, one activated.
func addInSession(t *testing.T) (*Router, *app.Session, *FakeWireAddIn) {
	t.Helper()
	s := app.NewSession()
	active := &FakeWireAddIn{id: "com.wire.active"}
	idle := &FakeWireAddIn{id: "com.wire.idle"}
	for _, a := range []*FakeWireAddIn{active, idle} {
		if err := s.AddIns().Register(a); err != nil {
			t.Fatalf("register %s: %v", a.id, err)
		}
	}
	if err := s.AddIns().Activate(s, active.id); err != nil {
		t.Fatalf("activate: %v", err)
	}
	return New(opregistry.Default()), s, active
}

func TestAddInsListAndGet(t *testing.T) {
	t.Parallel()
	r, s, _ := addInSession(t)
	var lst wire.ListAddInsResult
	call(t, r, s, "addins.list", "{}", &lst)
	if len(lst.AddIns) != 2 {
		t.Fatalf("list has %d entries, want 2", len(lst.AddIns))
	}
	if lst.AddIns[0].ID != "com.wire.active" || !lst.AddIns[0].Activated {
		t.Errorf("first entry = %+v, want activated com.wire.active", lst.AddIns[0])
	}
	if lst.AddIns[1].Activated {
		t.Errorf("idle entry reports activated")
	}

	var info wire.AddInInfo
	call(t, r, s, "addins.get", `{"id":"com.wire.idle"}`, &info)
	if info.DisplayName != "Wire Fake" || !info.HasAutomation {
		t.Errorf("get = %+v, want manifest identity and hasAutomation", info)
	}
}

func TestAddInsLifecycleOverWire(t *testing.T) {
	t.Parallel()
	r, s, _ := addInSession(t)
	call(t, r, s, "addins.activate", `{"id":"com.wire.idle"}`, nil)
	if !s.AddIns().IsActive("com.wire.idle") {
		t.Fatal("addins.activate did not activate")
	}
	call(t, r, s, "addins.deactivate", `{"id":"com.wire.idle"}`, nil)
	if s.AddIns().IsActive("com.wire.idle") {
		t.Fatal("addins.deactivate did not deactivate")
	}
}

func TestAddInsSetLoadBehaviorOverWire(t *testing.T) {
	t.Parallel()
	r, s, _ := addInSession(t)
	call(t, r, s, "addins.setLoadBehavior", `{"id":"com.wire.idle","loadBehavior":2}`, nil)
	if got := s.AddIns().LoadBehavior("com.wire.idle"); got != types.LoadDisabled {
		t.Fatalf("LoadBehavior = %v, want disabled", got)
	}
	if _, err := r.Handle(s, "addins.activate", []byte(`{"id":"com.wire.idle"}`)); err == nil {
		t.Fatal("activating a disabled add-in over the wire should fail")
	}
}

// TestAddInsCallAutomationOverWire is the #252 acceptance at the wire level: the
// caller reaches another add-in's automation surface via addins.callAutomation.
func TestAddInsCallAutomationOverWire(t *testing.T) {
	t.Parallel()
	r, s, _ := addInSession(t)
	var res wire.CallAddInAutomationResult
	call(t, r, s, "addins.callAutomation",
		`{"id":"com.wire.active","method":"solve","args":{"n":3}}`, &res)
	if string(res.Result) != `{"method":"solve","args":{"n":3}}` {
		t.Errorf("result = %s, want the target's echo", res.Result)
	}

	if _, err := r.Handle(s, "addins.callAutomation",
		[]byte(`{"id":"com.wire.idle","method":"solve"}`)); err == nil {
		t.Error("automation on an inactive add-in should fail over the wire")
	}
}

func TestClientAppsOverWire(t *testing.T) {
	t.Parallel()
	r, s, _ := addInSession(t)
	var reg wire.RegisterClientApplicationResult
	call(t, r, s, "clientApps.register", `{"name":"acme-pipeline"}`, &reg)
	if reg.ID == 0 {
		t.Fatal("register returned id 0")
	}
	var lst wire.ListClientApplicationsResult
	call(t, r, s, "clientApps.list", "{}", &lst)
	if len(lst.Clients) != 1 || lst.Clients[0].Name != "acme-pipeline" {
		t.Fatalf("list = %+v, want one acme-pipeline", lst.Clients)
	}
	call(t, r, s, "clientApps.unregister", fmt.Sprintf(`{"id":%d}`, reg.ID), nil)
	call(t, r, s, "clientApps.list", "{}", &lst)
	if len(lst.Clients) != 0 {
		t.Fatalf("list after unregister = %+v, want empty", lst.Clients)
	}
}
