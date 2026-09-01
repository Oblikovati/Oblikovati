// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestInteractionStateIdleVsInTransaction checks interaction.state reports idle on a fresh
// session and flags Busy/InTransaction once a bounded transaction is open — the signal a
// collaboration add-in gates remote edits on (ADR-0005).
func TestInteractionStateIdleVsInTransaction(t *testing.T) {
	t.Parallel()
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

// TestInteractionSetNotice checks an add-in can post a transient status-bar message, so a
// collaboration add-in's connection state is visible to the user (not just in logs).
func TestInteractionSetNotice(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	call(t, r, s, "interaction.setNotice", `{"message":"Meeting: connected"}`, nil)
	if s.Notice() != "Meeting: connected" {
		t.Errorf("notice = %q, want %q", s.Notice(), "Meeting: connected")
	}
}
