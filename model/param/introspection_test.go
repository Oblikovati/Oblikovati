// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

// TestRenamedFlagOnlyForModelParameters: renaming a model parameter raises Renamed;
// renaming a user parameter (whose name is authored, not generated) does not (#1853).
func TestRenamedFlagOnlyForModelParameters(t *testing.T) {
	ps := NewParameters()
	m, _ := ps.AddModelParameter("d0", "1 cm")
	u, _ := ps.AddUserParameter("width", "2 cm")
	if m.Renamed() || u.Renamed() {
		t.Fatalf("fresh parameters must not be renamed (model %v, user %v)", m.Renamed(), u.Renamed())
	}
	if err := ps.Rename(m.ID(), "length"); err != nil {
		t.Fatalf("rename model: %v", err)
	}
	if err := ps.Rename(u.ID(), "breadth"); err != nil {
		t.Fatalf("rename user: %v", err)
	}
	if !m.Renamed() {
		t.Error("renamed model parameter should report Renamed() = true")
	}
	if u.Renamed() {
		t.Error("renamed user parameter should stay Renamed() = false (authored name)")
	}
}

// TestAutoModelParameterIsBuiltIn confirms BuiltIn distinguishes an auto-generated
// feature-dimension backing param from a user-created one (#1853 acceptance).
func TestAutoModelParameterIsBuiltIn(t *testing.T) {
	ps := NewParameters()
	auto, _ := ps.AddAutoModelParameter("3 cm")
	user, _ := ps.AddUserParameter("w", "3 cm")
	if !auto.BuiltIn() {
		t.Error("auto model parameter should be BuiltIn() = true")
	}
	if user.BuiltIn() {
		t.Error("user parameter should be BuiltIn() = false")
	}
}

// TestDisabledActionTypesRoundTrip: the mask is settable and read back exactly.
func TestDisabledActionTypesRoundTrip(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("w", "3 cm")
	if p.DisabledActionTypes() != ActionNone {
		t.Fatalf("fresh parameter mask = %v, want ActionNone", p.DisabledActionTypes())
	}
	p.SetDisabledActionTypes(ActionRename | ActionDelete)
	if got := p.DisabledActionTypes(); got != ActionRename|ActionDelete {
		t.Errorf("mask = %v, want rename|delete", got)
	}
}
