// SPDX-License-Identifier: GPL-2.0-only

package attr

import "testing"

func TestAttributeSetCRUD(t *testing.T) {
	s := newAttributeSet("acme")
	if s.Name() != "acme" {
		t.Errorf("Name = %q", s.Name())
	}
	a := s.Put("width", IntValue(10))
	if a.Name() != "width" || a.ValueType() != Integer {
		t.Errorf("Put returned %+v", a)
	}
	// Put with an existing name updates in place (same pointer).
	if s.Put("width", IntValue(20)) != a {
		t.Error("Put created a new attribute instead of updating")
	}
	if got, _ := a.Value().Int(); got != 20 {
		t.Errorf("value after update = %d, want 20", got)
	}
	a.SetValue(StringValue("x"))
	if a.ValueType() != String {
		t.Error("SetValue did not change the type")
	}
	if _, ok := s.Attribute("width"); !ok || s.Count() != 1 {
		t.Error("Attribute/Count wrong")
	}
	if !s.Remove("width") || s.Count() != 0 || s.Remove("width") {
		t.Error("Remove behavior wrong")
	}
}

func TestAttributeSetOrdering(t *testing.T) {
	s := newAttributeSet("s")
	s.Put("c", IntValue(1))
	s.Put("a", IntValue(2))
	s.Put("b", IntValue(3))
	got := s.Attributes()
	if len(got) != 3 || got[0].Name() != "c" || got[1].Name() != "a" || got[2].Name() != "b" {
		t.Errorf("Attributes order = %v, want insertion order c,a,b", names(got))
	}
}

func names(attrs []*Attribute) []string {
	out := make([]string, len(attrs))
	for i, a := range attrs {
		out[i] = a.Name()
	}
	return out
}

func TestAttributeSetsCRUD(t *testing.T) {
	ss := newAttributeSets()
	s := ss.Set("one")
	if ss.Set("one") != s {
		t.Error("Set did not return the existing set")
	}
	ss.Set("two")
	if ss.Count() != 2 {
		t.Errorf("Count = %d, want 2", ss.Count())
	}
	if got := ss.Names(); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("Names = %v, want [one two]", got)
	}
	if _, ok := ss.Lookup("one"); !ok {
		t.Error("Lookup missed an existing set")
	}
	if _, ok := ss.Lookup("nope"); ok {
		t.Error("Lookup found a nonexistent set")
	}
	if !ss.Remove("one") || ss.Count() != 1 || ss.Remove("one") {
		t.Error("Remove behavior wrong")
	}
	if len(ss.Sets()) != 1 {
		t.Error("Sets count wrong after remove")
	}
}
