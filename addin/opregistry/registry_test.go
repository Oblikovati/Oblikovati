// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"testing"

	"github.com/Oblikovati/oblikovati/app"
)

func TestDefaultHasExtrude(t *testing.T) {
	r := Default()
	d, ok := r.ByName("extrude")
	if !ok {
		t.Fatal("default registry missing extrude")
	}
	if d.Summary == "" || len(d.Schema) == 0 || d.Apply == nil {
		t.Fatalf("extrude descriptor incomplete: %+v", d)
	}
	// Schema must be valid JSON so it can be served as-is to an LLM.
	var schema map[string]any
	if err := json.Unmarshal(d.Schema, &schema); err != nil {
		t.Fatalf("extrude schema is not valid JSON: %v", err)
	}
}

func TestRegisterValidates(t *testing.T) {
	r := New()
	if err := r.Register(&OperationDescriptor{Name: "", Apply: dummyApply}); err == nil {
		t.Error("expected error for empty name")
	}
	if err := r.Register(&OperationDescriptor{Name: "x"}); err == nil {
		t.Error("expected error for missing Apply")
	}
	if err := r.Register(&OperationDescriptor{Name: "x", Apply: dummyApply}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := r.Register(&OperationDescriptor{Name: "x", Apply: dummyApply}); err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestAllInRegistrationOrder(t *testing.T) {
	r := New()
	_ = r.Register(&OperationDescriptor{Name: "a", Apply: dummyApply})
	_ = r.Register(&OperationDescriptor{Name: "b", Apply: dummyApply})
	all := r.All()
	if len(all) != 2 || all[0].Name != "a" || all[1].Name != "b" {
		t.Fatalf("All order = %v, want [a b]", names(all))
	}
}

func names(ds []*OperationDescriptor) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Name
	}
	return out
}

func dummyApply(_ *app.Session, _ json.RawMessage) (json.RawMessage, error) { return nil, nil }
