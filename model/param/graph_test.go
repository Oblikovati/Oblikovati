// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"strings"
	"testing"
)

// mustAdd is a test helper that adds a user parameter and fails on error.
func mustAdd(t *testing.T, ps *Parameters, name, expr string) *Parameter {
	t.Helper()
	p, err := ps.AddUserParameter(name, expr)
	if err != nil {
		t.Fatalf("AddUserParameter(%q): %v", name, err)
	}
	return p
}

// healthyValue is a test helper: it asserts a parameter is healthy and returns
// its database value.
func healthyValue(t *testing.T, p *Parameter) float64 {
	t.Helper()
	if !p.Health().OK() {
		t.Fatalf("%s unhealthy: %+v", p.Name(), p.Health())
	}
	return p.Value().Value
}

func TestExpressionReferencesEvaluate(t *testing.T) {
	ps := NewParameters()
	w, _ := ps.AddUserParameter("width", "10 cm")
	h, _ := ps.AddUserParameter("height", "2 * width") // 20 cm
	if got := healthyValue(t, h); !approxScalar(got, 20) {
		t.Errorf("height = %v, want 20", got)
	}
	// Edges are recorded both ways.
	if deps := ps.Dependents(w.ID()); len(deps) != 1 || deps[0] != h.ID() {
		t.Errorf("width.Dependents = %v, want [height]", deps)
	}
	if drv := ps.DrivenBy(h.ID()); len(drv) != 1 || drv[0] != w.ID() {
		t.Errorf("height.DrivenBy = %v, want [width]", drv)
	}
}

func TestChangePropagatesToDependentsOnly(t *testing.T) {
	ps := NewParameters()
	w, _ := ps.AddUserParameter("width", "10 cm")
	h, _ := ps.AddUserParameter("height", "2 * width")
	indep, _ := ps.AddUserParameter("indep", "3 cm")
	// Change width → height (its only dependent) must update; indep must not.
	if err := ps.SetExpression(w.ID(), "20 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	if got := healthyValue(t, h); !approxScalar(got, 40) {
		t.Errorf("height after width change = %v, want 40", got)
	}
	if got := healthyValue(t, indep); !approxScalar(got, 3) {
		t.Errorf("independent parameter changed to %v, want 3", got)
	}
}

func TestTransitiveChainRecomputes(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "2 cm")
	mustAdd(t, ps, "b", "a * 2")              // 4
	c, _ := ps.AddUserParameter("c", "b + a") // 6
	if err := ps.SetValue(a.ID(), Q(5, Length)); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	// b = 10, c = b + a = 15.
	if got := healthyValue(t, c); !approxScalar(got, 15) {
		t.Errorf("c after a change = %v, want 15", got)
	}
}

func TestCycleRejected(t *testing.T) {
	ps := NewParameters()
	a, _ := ps.AddUserParameter("a", "2 cm")
	b, _ := ps.AddUserParameter("b", "a + 1 cm")
	// Make a depend on b → a→b→a cycle.
	err := ps.SetExpression(a.ID(), "b + 1 cm")
	if err == nil {
		t.Fatal("cyclic expression should be rejected")
	}
	if _, ok := err.(*CycleError); !ok {
		t.Errorf("error type = %T, want *CycleError", err)
	}
	if a.Health().Status != Failed {
		t.Errorf("a health = %+v, want Failed (sick, not crashed)", a.Health())
	}
	// b is untouched and still healthy.
	if !b.Health().OK() {
		t.Errorf("b should remain healthy, got %+v", b.Health())
	}
}

func TestUndefinedReferenceGoesSick(t *testing.T) {
	ps := NewParameters()
	p, err := ps.AddUserParameter("p", "missing + 1 cm")
	if err != nil {
		t.Fatalf("add should succeed (undefined ref is health, not error): %v", err)
	}
	if p.Health().Status != Failed {
		t.Errorf("health = %+v, want Failed on undefined reference", p.Health())
	}
}

func TestForwardReferenceResolvesWhenDriverAdded(t *testing.T) {
	ps := NewParameters()
	dep, _ := ps.AddUserParameter("dep", "base * 2") // base not yet defined → sick
	if dep.Health().Status != Failed {
		t.Fatalf("dep should be sick before base exists, got %+v", dep.Health())
	}
	mustAdd(t, ps, "base", "5 cm") // adding base should resolve dep
	if got := healthyValue(t, dep); !approxScalar(got, 10) {
		t.Errorf("dep after base added = %v, want 10", got)
	}
}

func TestRenamePreservesEdgesAndRewritesDisplay(t *testing.T) {
	ps := NewParameters()
	w, _ := ps.AddUserParameter("width", "10 cm")
	h, _ := ps.AddUserParameter("height", "2 * width + 5 mm")
	if err := ps.Rename(w.ID(), "w"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	// Edge preserved: height still depends on the (renamed) width.
	if drv := ps.DrivenBy(h.ID()); len(drv) != 1 || drv[0] != w.ID() {
		t.Errorf("edge lost after rename: DrivenBy = %v", drv)
	}
	// Display rewritten, units untouched.
	if h.Expression() != "2 * w + 5 mm" {
		t.Errorf("display = %q, want \"2 * w + 5 mm\"", h.Expression())
	}
	// Still evaluates correctly and an edit that rebinds still resolves "w".
	if err := ps.SetExpression(w.ID(), "20 cm"); err != nil {
		t.Fatalf("SetExpression after rename: %v", err)
	}
	if got := healthyValue(t, h); !approxScalar(got, 40.5) { // 2*20 + 0.5
		t.Errorf("height after rename+change = %v, want 40.5", got)
	}
}

// Deleting an in-use driver is refused with the blockers named — the aggregate
// invariant every driver (wire, UI) now shares (#1612, audit B1). The old
// behavior on this path ("dependents go sick") survives only for the owner
// cascade, checked below.
func TestDeleteRefusesInUseDriver(t *testing.T) {
	ps := NewParameters()
	w, _ := ps.AddUserParameter("width", "10 cm")
	h, _ := ps.AddUserParameter("height", "2 * width")
	err := ps.Delete(w.ID())
	if err == nil || !strings.Contains(err.Error(), "height") {
		t.Fatalf("Delete(in-use) = %v, want a refusal naming the blocker \"height\"", err)
	}
	if _, ok := ps.ByID(w.ID()); !ok {
		t.Error("a refused delete must leave the parameter in place")
	}
	if h.Health().Status == Failed {
		t.Errorf("dependent health after refused delete = %+v, want untouched", h.Health())
	}
}

// TestDeleteForOwnerSickensDependents keeps the owner-cascade contract: a
// dimension tearing down its own model parameter bypasses the guard, and
// former dependents go sick on the lost reference.
func TestDeleteForOwnerSickensDependents(t *testing.T) {
	ps := NewParameters()
	w, _ := ps.AddModelParameter("d0", "10 cm")
	h, _ := ps.AddUserParameter("height", "2 * d0")
	if err := ps.DeleteForOwner(w.ID()); err != nil {
		t.Fatalf("DeleteForOwner: %v", err)
	}
	if h.Health().Status != Failed {
		t.Errorf("dependent health after owner delete = %+v, want Failed", h.Health())
	}
}

func TestInUseFollowsDependentsAndModelKind(t *testing.T) {
	ps := NewParameters()
	free, _ := ps.AddUserParameter("free", "1 mm")
	driver, _ := ps.AddUserParameter("od", "10 mm")
	dependent, _ := ps.AddUserParameter("wall", "od / 5")
	dim, _ := ps.AddModelParameter("d0", "2 mm")

	if ps.InUse(free.ID()) {
		t.Error("an unreferenced user parameter is not in use")
	}
	if !ps.InUse(driver.ID()) {
		t.Error("a parameter read by another is in use")
	}
	if ps.InUse(dependent.ID()) {
		t.Error("a leaf dependent is not in use")
	}
	// Model parameters belong to a feature dimension, so they are always in use.
	if !ps.InUse(dim.ID()) {
		t.Error("a model parameter is in use by construction")
	}
}

// TestDeleteRefusesOwnedModelParameter: with no dependents, a model parameter's
// blocker is its owning feature dimension; DeleteForOwner errors on unknown ids.
func TestDeleteRefusesOwnedModelParameter(t *testing.T) {
	ps := NewParameters()
	dim, _ := ps.AddModelParameter("d0", "2 mm")
	err := ps.Delete(dim.ID())
	if err == nil || !strings.Contains(err.Error(), "feature dimension") {
		t.Fatalf("Delete(model param) = %v, want a refusal naming its feature dimension", err)
	}
	if err := ps.DeleteForOwner(dim.ID() + 999); err == nil {
		t.Error("DeleteForOwner(unknown id) must error")
	}
}
