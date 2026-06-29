// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"slices"
	"strings"
	"testing"
)

// gearSource is the resolver's view of a source document with two linkable
// numeric parameters.
func gearSource() []SourceParameterValue {
	return []SourceParameterValue{
		{Name: "module", Value: Quantity{Value: 0.2, Unit: Length}},
		{Name: "teeth", Value: Quantity{Value: 24, Unit: Unitless}},
	}
}

func TestAddDerivedTableProducesReadOnlyParameters(t *testing.T) {
	ps := NewParameters()
	table, err := ps.AddDerivedTable("gears.obk", []string{"module"}, gearSource())
	if err != nil {
		t.Fatalf("AddDerivedTable: %v", err)
	}
	if table.SourceDocument() != "gears.obk" || !slices.Equal(table.Linked(), []string{"module"}) {
		t.Errorf("table = %+v, want gears.obk linking module", table)
	}
	p, ok := ps.ByName("module")
	if !ok || p.Kind() != DerivedParam || !approxScalar(p.Value().Value, 0.2) {
		t.Fatalf("derived parameter = %+v, want read-only module = 0.2", p)
	}
	if err := p.SetExpression("1 mm"); err == nil {
		t.Error("a derived parameter must be read-only")
	}

	if _, err := ps.AddDerivedTable("gears.obk", []string{"bogus"}, gearSource()); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Errorf("unknown link err = %v, want a rejection naming bogus", err)
	}
	if _, err := ps.AddDerivedTable("", nil, nil); err == nil {
		t.Error("empty source document must be rejected")
	}
	// A second table linking the same name clashes with the produced parameter.
	if _, err := ps.AddDerivedTable("other.obk", []string{"module"}, gearSource()); err == nil {
		t.Error("a link clashing with an existing parameter must be rejected")
	}
}

func TestSetDerivedTableLinkedDiffsTheSubset(t *testing.T) {
	ps := NewParameters()
	table, _ := ps.AddDerivedTable("gears.obk", []string{"module"}, gearSource())

	if err := ps.SetDerivedTableLinked(table.ID(), []string{"teeth"}, gearSource()); err != nil {
		t.Fatalf("SetDerivedTableLinked: %v", err)
	}
	if _, ok := ps.ByName("module"); ok {
		t.Error("unlinking must delete the produced derived parameter")
	}
	if p, ok := ps.ByName("teeth"); !ok || p.Kind() != DerivedParam {
		t.Error("newly linked name must gain a derived parameter")
	}
}

func TestSyncDerivedTablePropagatesAndSickens(t *testing.T) {
	ps := NewParameters()
	table, _ := ps.AddDerivedTable("gears.obk", []string{"module"}, gearSource())
	// A local parameter computing on the derived value.
	dependent, _ := ps.AddUserParameter("pitch", "module * 24")
	if !approxScalar(dependent.Value().Value, 4.8) {
		t.Fatalf("dependent = %v, want 4.8 (0.2 * 24)", dependent.Value().Value)
	}

	// A source value change flows through to dependents.
	changed := []SourceParameterValue{{Name: "module", Value: Quantity{Value: 0.25, Unit: Length}}}
	if err := ps.SyncDerivedTable(table.ID(), changed, true); err != nil {
		t.Fatalf("SyncDerivedTable: %v", err)
	}
	if !approxScalar(dependent.Value().Value, 6) {
		t.Errorf("dependent after sync = %v, want 6 (0.25 * 24)", dependent.Value().Value)
	}

	// A removed source parameter sickens its derived counterpart but keeps the
	// last value so the model still computes.
	if err := ps.SyncDerivedTable(table.ID(), nil, true); err != nil {
		t.Fatalf("sync with removed source: %v", err)
	}
	module, _ := ps.ByName("module")
	if module.Health().OK() || !approxScalar(module.Value().Value, 0.25) {
		t.Errorf("after removal: health=%+v value=%v, want sick at last value 0.25", module.Health(), module.Value().Value)
	}
	if table.Health().OK() || !strings.Contains(table.Health().Reason, "module") {
		t.Errorf("table health = %+v, want failed naming module", table.Health())
	}

	// The source coming back restores health.
	if err := ps.SyncDerivedTable(table.ID(), gearSource(), true); err != nil {
		t.Fatalf("recovery sync: %v", err)
	}
	if !module.Health().OK() || !table.Health().OK() {
		t.Errorf("after recovery: param=%+v table=%+v, want both healthy", module.Health(), table.Health())
	}

	// An unreachable source document sickens the whole table.
	_ = ps.SyncDerivedTable(table.ID(), nil, false)
	if table.Health().OK() || module.Health().OK() {
		t.Error("unreachable source must sicken the table and its parameters")
	}
}

func TestDeleteDerivedTableRules(t *testing.T) {
	ps := NewParameters()
	table, _ := ps.AddDerivedTable("gears.obk", []string{"module", "teeth"}, gearSource())

	owned, _ := ps.AddDerivedTable("hub.obk", nil, []SourceParameterValue{})
	owned.MarkOwnedByFeature()
	if err := ps.DeleteDerivedTable(owned.ID()); err == nil || !strings.Contains(err.Error(), "component") {
		t.Errorf("component-owned delete err = %v, want a refusal pointing at the component", err)
	}

	if err := ps.DeleteDerivedTable(table.ID()); err != nil {
		t.Fatalf("DeleteDerivedTable: %v", err)
	}
	if _, ok := ps.ByName("module"); ok {
		t.Error("deleting a table must delete its derived parameters")
	}
	if err := ps.DeleteDerivedTable(table.ID()); err == nil {
		t.Error("deleting an unknown table must error")
	}
}

func TestRestoreDerivedTableReconnectsByName(t *testing.T) {
	ps := NewParameters()
	// The parameter list restores first (as ApplyRecipe does), then the table.
	_, _ = ps.AddDerivedParameter("module", Quantity{Value: 0.2, Unit: Length})
	if err := ps.RestoreDerivedTable(7, "gears.obk", []string{"module"}, true); err != nil {
		t.Fatalf("RestoreDerivedTable: %v", err)
	}
	table, ok := ps.DerivedTableByID(7)
	if !ok || !table.OwnedByFeature() {
		t.Fatalf("restored table = %+v, want id 7 owned by feature", table)
	}
	// The reconnected link syncs like a created one.
	changed := []SourceParameterValue{{Name: "module", Value: Quantity{Value: 0.3, Unit: Length}}}
	_ = ps.SyncDerivedTable(7, changed, true)
	p, _ := ps.ByName("module")
	if !approxScalar(p.Value().Value, 0.3) {
		t.Errorf("synced reconnected value = %v, want 0.3", p.Value().Value)
	}
	// The persistent counter never reissues a restored id.
	next, _ := ps.AddDerivedTable("other.obk", nil, nil)
	if next.ID() <= 7 {
		t.Errorf("next table id = %d, want above the restored 7", next.ID())
	}
}

// TestDerivedTableReferences checks each produced derived parameter reports its source
// document and source-parameter name — the reference API's ReferencedEntity (M39-F05, #1561).
func TestDerivedTableReferences(t *testing.T) {
	ps := NewParameters()
	table, err := ps.AddDerivedTable("gears.obk", []string{"module", "teeth"}, gearSource())
	if err != nil {
		t.Fatalf("AddDerivedTable: %v", err)
	}
	refs := table.References()
	if len(refs) != 2 {
		t.Fatalf("references = %d, want 2 (one per produced derived parameter)", len(refs))
	}
	for _, r := range refs {
		if r.SourceDocument != "gears.obk" || r.DerivedName != r.SourceName {
			t.Errorf("reference = %+v, want gears.obk with matching derived/source name", r)
		}
	}
	if refs[0].DerivedName != "module" || refs[1].DerivedName != "teeth" {
		t.Errorf("reference order = [%s,%s], want [module,teeth] (link order)", refs[0].DerivedName, refs[1].DerivedName)
	}
}

// TestDerivedTableReferencesSkipUnproduced checks a linked-but-not-yet-produced name (one
// restored awaiting a sync) yields no reference until its derived parameter exists.
func TestDerivedTableReferencesSkipUnproduced(t *testing.T) {
	ps := NewParameters()
	// RestoreDerivedTable records a linked name without producing a parameter (no source yet).
	if err := ps.RestoreDerivedTable(1, "gears.obk", []string{"module"}, false); err != nil {
		t.Fatalf("RestoreDerivedTable: %v", err)
	}
	table, _ := ps.DerivedTableByID(1)
	if got := table.References(); len(got) != 0 {
		t.Errorf("references for an unproduced link = %+v, want none", got)
	}
}
