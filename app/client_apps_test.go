// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestClientAppsRegisterListUnregister(t *testing.T) {
	t.Parallel()
	reg := NewSession().ClientApps()
	idA, err := reg.Register("acme-pipeline")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	idB, err := reg.Register("qa-harness")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if idA == idB {
		t.Fatalf("ids collide: %d", idA)
	}

	lst := reg.List()
	if len(lst) != 2 || lst[0].Name != "acme-pipeline" || lst[1].Name != "qa-harness" {
		t.Fatalf("List = %+v, want registration order acme-pipeline, qa-harness", lst)
	}

	if err := reg.Unregister(idA); err != nil {
		t.Fatalf("Unregister: %v", err)
	}
	if lst := reg.List(); len(lst) != 1 || lst[0].ID != idB {
		t.Errorf("List after unregister = %+v, want only qa-harness", lst)
	}
}

func TestClientAppsRejectEmptyNameAndUnknownID(t *testing.T) {
	t.Parallel()
	reg := NewClientApplicationRegistry()
	if _, err := reg.Register(""); err == nil {
		t.Error("Register(\"\") should fail")
	}
	if err := reg.Unregister(99); err == nil {
		t.Error("Unregister(99) should fail for an unknown id")
	}
}
