// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestNetworkToolParamsAndGate(t *testing.T) {
	tool := NewNetworkTool()
	if tool.Name() != "Network Surface" || tool.Prompt(nil) == "" {
		t.Error("network tool should name and prompt")
	}
	if tool.CanCommit() {
		t.Error("network tool should not be committable with no curves")
	}
	p := tool.Params()
	if len(p.Bools) != 1 {
		t.Fatalf("network tool should expose the V-pick toggle, got %d bools", len(p.Bools))
	}
	p.Bools[0].Set(true)
	if !p.Bools[0].Get() {
		t.Error("V-pick toggle get/set round-trip mismatch")
	}
}

func TestNetworkViaRibbonCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 2)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Surface.Network"); err != nil {
		t.Fatalf("execute Surface.Network: %v", err)
	}
	if got := s.ActiveTool().Name(); got != "Network Surface" {
		t.Errorf("Surface.Network started tool %q, want Network Surface", got)
	}
}
