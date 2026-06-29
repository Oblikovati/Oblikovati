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

// Derived parameter tables over the wire (M02-F06, Oblikovati#605).

// derivedWireSession is a seeded session plus a second, active part document
// ready to derive from the seeded "test.obk" (which holds "width" = 4 cm).
func derivedWireSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := seededSession(t)
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	return r, s
}

func TestDerivedTablesLifecycleOverWire(t *testing.T) {
	r, s := derivedWireSession(t)

	var info wire.DerivedParameterTableInfo
	call(t, r, s, "parameters.derivedTables.add", `{"sourceDocument":"test.obk","linked":["width"]}`, &info)
	if info.SourceDocument != "test.obk" || !slices.Equal(info.Linked, []string{"width"}) || info.Health != "" {
		t.Fatalf("added table = %+v, want a healthy width link", info)
	}
	if !slices.Contains(info.Available, "width") {
		t.Errorf("available = %v, want the source candidates listed", info.Available)
	}
	// The produced derived parameter answers member-level reads.
	if d := getDetail(t, r, s, "width"); d.Kind != "derived" {
		t.Errorf("derived parameter kind = %q, want derived", d.Kind)
	}

	var list wire.ListDerivedParameterTablesResult
	call(t, r, s, "parameters.derivedTables.list", "{}", &list)
	if len(list.Tables) != 1 || list.Tables[0].ID != info.ID {
		t.Fatalf("tables = %+v, want the one added table", list.Tables)
	}

	var relinked wire.DerivedParameterTableInfo
	call(t, r, s, "parameters.derivedTables.setLinked", `{"id":1}`, &relinked)
	if len(relinked.Linked) != 0 {
		t.Errorf("relinked = %+v, want an empty subset", relinked)
	}
	if _, err := r.Handle(s, "parameters.getDetail", []byte(`{"name":"width"}`)); err == nil {
		t.Error("unlinking must delete the produced derived parameter")
	}

	call(t, r, s, "parameters.derivedTables.delete", `{"id":1}`, nil)
	// A fresh var: an omitted "tables" would merge over a reused struct.
	var after wire.ListDerivedParameterTablesResult
	call(t, r, s, "parameters.derivedTables.list", "{}", &after)
	if len(after.Tables) != 0 {
		t.Errorf("tables after delete = %+v, want none", after.Tables)
	}
}

func TestDerivedTablesRejectBadArgs(t *testing.T) {
	r, s := derivedWireSession(t)
	for method, args := range map[string]string{
		"parameters.derivedTables.add":       `{"sourceDocument":"missing.obk"}`,
		"parameters.derivedTables.setLinked": `{"id":99}`,
		"parameters.derivedTables.delete":    `{"id":99}`,
	} {
		if _, err := r.Handle(s, method, []byte(args)); err == nil {
			t.Errorf("%s(%s) must be rejected", method, args)
		}
	}
}

// TestDerivedTableMutationsBroadcast checks the mutations emit edit.committed
// and the list read does not.
func TestDerivedTableMutationsBroadcast(t *testing.T) {
	r, s := derivedWireSession(t)
	var methods []string
	sub := event.Subscribe(s.Events(), event.After, func(_ event.Context, e app.EditCommitted) event.Outcome {
		methods = append(methods, e.Method)
		return event.Continue()
	})
	defer sub.Cancel()

	call(t, r, s, "parameters.derivedTables.add", `{"sourceDocument":"test.obk","linked":["width"]}`, nil)
	call(t, r, s, "parameters.derivedTables.list", "{}", nil)
	call(t, r, s, "parameters.derivedTables.setLinked", `{"id":1,"linked":[]}`, nil)
	call(t, r, s, "parameters.derivedTables.delete", `{"id":1}`, nil)

	want := []string{
		wire.MethodParametersDerivedTablesAdd,
		wire.MethodParametersDerivedTablesSetLinked,
		wire.MethodParametersDerivedTablesDelete,
	}
	if !slices.Equal(methods, want) {
		t.Errorf("edit.committed methods = %v, want %v (and none for list)", methods, want)
	}
}

// TestDerivedValueFollowsSourceOverWire drives the wire end of the
// update-on-source-change behavior: editing the source parameter over the
// wire pushes the new value into the deriving document.
func TestDerivedValueFollowsSourceOverWire(t *testing.T) {
	r, s := derivedWireSession(t)
	call(t, r, s, "parameters.derivedTables.add", `{"sourceDocument":"test.obk","linked":["width"]}`, nil)
	deriving := s.ActiveDocument()

	// Activate the source and move width over the wire.
	src, ok := s.Workspace().ByName("test.obk")
	if !ok {
		t.Fatal("seeded source document missing")
	}
	if err := s.Workspace().SetActiveDocument(src); err != nil {
		t.Fatalf("activate source: %v", err)
	}
	call(t, r, s, "parameters.set", `{"name":"width","expression":"6 cm"}`, nil)

	// Back on the deriving document, the derived value followed.
	if err := s.Workspace().SetActiveDocument(deriving); err != nil {
		t.Fatalf("re-activate deriving doc: %v", err)
	}
	if d := getDetail(t, r, s, "width"); !strings.Contains(d.Value, "6") {
		t.Errorf("derived width after source edit = %q, want it at 6 cm", d.Value)
	}
}

// TestDerivedTableInfoReferencesAndProvenance checks the wire info carries the per-derived
// parameter references and the (false, for a user-created table) reference-component
// provenance (M39-F05, #1561).
func TestDerivedTableInfoReferencesAndProvenance(t *testing.T) {
	r, s := derivedWireSession(t)
	var info wire.DerivedParameterTableInfo
	call(t, r, s, "parameters.derivedTables.add", `{"sourceDocument":"test.obk","linked":["width"]}`, &info)
	if len(info.References) != 1 {
		t.Fatalf("references = %+v, want one", info.References)
	}
	ref := info.References[0]
	if ref.Parameter != "width" || ref.SourceDocument != "test.obk" || ref.SourceParameter != "width" {
		t.Errorf("reference = %+v, want width linked to test.obk.width", ref)
	}
	if info.HasReferenceComponent || info.ReferenceComponent != "" {
		t.Errorf("a user-created derived table is not component-owned: has=%v comp=%q", info.HasReferenceComponent, info.ReferenceComponent)
	}
}
