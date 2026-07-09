// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

// Parameter type conversion (#1850): ConvertTo{User,Model,Reference} changes a parameter's
// category in place, keeping its identity and dependency edges; a built-in/auto or derived
// parameter, or an unsupported target, is refused.

// TestConvertPreservesIdentityAndEdges: converting "od" user→model keeps its name and expression,
// and a dependent "od/2" still tracks it (the edge binds by id, so re-kinding is transparent). It
// converts back to user cleanly.
func TestConvertPreservesIdentityAndEdges(t *testing.T) {
	ps := NewParameters()
	od, _ := ps.AddUserParameter("od", "10 cm")
	half, _ := ps.AddUserParameter("half", "od / 2")

	if err := ps.Convert(od.ID(), ModelParam); err != nil {
		t.Fatalf("user→model: %v", err)
	}
	if od.Kind() != ModelParam || od.Name() != "od" || od.Expression() != "10 cm" {
		t.Errorf("converted param lost identity: kind=%v name=%q expr=%q", od.Kind(), od.Name(), od.Expression())
	}
	// The dependent edge survived: driving od re-evaluates half.
	if err := ps.SetExpression(od.ID(), "20 cm"); err != nil {
		t.Fatalf("set od after convert: %v", err)
	}
	if got := half.Value().Value; got != 10 {
		t.Errorf("dependent half = %g cm, want 10 (edge lost across conversion)", got)
	}
	if err := ps.Convert(od.ID(), UserParam); err != nil || od.Kind() != UserParam {
		t.Errorf("model→user: err=%v kind=%v", err, od.Kind())
	}
}

// TestConvertToReferenceIsReadOnly: converting to reference makes the parameter read-only (a later
// SetExpression is refused); converting back to user restores editability.
func TestConvertToReferenceIsReadOnly(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("a", "5 cm")

	if err := ps.Convert(p.ID(), ReferenceParam); err != nil {
		t.Fatalf("user→reference: %v", err)
	}
	if p.Kind() != ReferenceParam {
		t.Fatalf("kind = %v, want reference", p.Kind())
	}
	if err := p.SetExpression("6 cm"); err == nil {
		t.Error("a reference parameter must refuse SetExpression")
	}
	if err := ps.Convert(p.ID(), UserParam); err != nil {
		t.Fatalf("reference→user: %v", err)
	}
	if err := p.SetExpression("6 cm"); err != nil {
		t.Errorf("a user parameter must accept SetExpression, got %v", err)
	}
}

// TestConvertBuiltInRejected: an auto-generated model parameter (a feature-dimension backing param)
// is built-in and refuses conversion.
func TestConvertBuiltInRejected(t *testing.T) {
	ps := NewParameters()
	d, _ := ps.AddAutoModelParameter("3 cm")
	if !d.BuiltIn() {
		t.Fatal("AddAutoModelParameter should mint a built-in parameter")
	}
	if err := ps.Convert(d.ID(), UserParam); err == nil {
		t.Error("a built-in/auto parameter must not convert")
	}
}

// TestConvertDerivedAndBadTargetRejected: a derived (cross-document) parameter cannot convert, and
// only user/model/reference are valid targets.
func TestConvertDerivedAndBadTargetRejected(t *testing.T) {
	ps := NewParameters()
	d, _ := ps.AddDerivedParameter("x", Q(1, Length))
	if err := ps.Convert(d.ID(), UserParam); err == nil {
		t.Error("a derived parameter must not convert")
	}
	p, _ := ps.AddUserParameter("a", "5 cm")
	if err := ps.Convert(p.ID(), DerivedParam); err == nil {
		t.Error("derived is not a valid conversion target")
	}
	if err := ps.Convert(ID(9999), UserParam); err == nil {
		t.Error("converting an unknown id should error")
	}
}
