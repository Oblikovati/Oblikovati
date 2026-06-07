// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati/addin/opregistry"
	"oblikovati/api/wire"
	"oblikovati/app"
)

// TestInteractionStateIdleVsInTransaction checks interaction.state reports idle on a fresh
// session and flags Busy/InTransaction once a bounded transaction is open — the signal a
// collaboration add-in gates remote edits on (ADR-0005).
func TestInteractionStateIdleVsInTransaction(t *testing.T) {
	r := New(opregistry.Default())
	s := app.NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}

	var st wire.InteractionState
	call(t, r, s, "interaction.state", "{}", &st)
	if st.Busy || st.InTransaction || st.ActiveTool != "" {
		t.Fatalf("fresh session should be idle, got %+v", st)
	}

	call(t, r, s, "transaction.begin", `{"label":"x"}`, nil)
	call(t, r, s, "interaction.state", "{}", &st)
	if !st.InTransaction || !st.Busy {
		t.Fatalf("inside a transaction state should be busy/inTransaction, got %+v", st)
	}
}
