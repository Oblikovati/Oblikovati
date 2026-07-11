// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/options"
)

// TestTangentChainSelectDefaultsOn checks the session-level default that new Fillet/Chamfer tools
// seed from is on out of the box (#1947), matching Inventor's tangent propagation.
func TestTangentChainSelectDefaultsOn(t *testing.T) {
	s := NewSession()
	if !s.TangentChainSelect() {
		t.Error("TangentChainSelect() should default to true (Inventor tangent propagation)")
	}
	f := NewFilletTool()
	s.StartTool(f)
	if !f.TangentChain() {
		t.Error("a new Fillet tool should seed Tangent chain = on from the session default")
	}
	ch := NewChamferTool()
	s.StartTool(ch)
	if !ch.TangentChain() {
		t.Error("a new Chamfer tool should seed Tangent chain = on from the session default")
	}
}

// TestTangentChainSelectPreferenceSeedsNewTools verifies that flipping the session preference is
// picked up by the next tool started (the preference is the seed, not a one-shot).
func TestTangentChainSelectPreferenceSeedsNewTools(t *testing.T) {
	s := NewSession()
	s.SetTangentChainSelect(false)
	f := NewFilletTool()
	s.StartTool(f)
	if f.TangentChain() {
		t.Error("after SetTangentChainSelect(false), a new Fillet tool should start with Tangent chain off")
	}
}

// TestTangentChainSelectPersists round-trips the preference through SetPartOptions (the persistence
// path the Preferences window and options.* wire methods use), mirroring chamfer flat-corners.
func TestTangentChainSelectPersists(t *testing.T) {
	s := NewSession()
	if err := s.SetPartOptions(options.Part{ChamferFlatCorners: true, TangentChainSelect: false}); err != nil {
		t.Fatalf("SetPartOptions: %v", err)
	}
	if s.TangentChainSelect() {
		t.Error("SetPartOptions{TangentChainSelect:false} should turn the session default off")
	}
	if got := s.Options().Part.TangentChainSelect; got {
		t.Errorf("Options().Part.TangentChainSelect = %v, want false (persisted)", got)
	}
}

// TestOptionsDefaultsTangentChainOn guards the on-disk defaults: a fresh options set (no stored
// file) has tangent-chain selection enabled, so Load-over-Defaults keeps it on for files that
// predate the key.
func TestOptionsDefaultsTangentChainOn(t *testing.T) {
	if !options.Defaults().Part.TangentChainSelect {
		t.Error("options.Defaults().Part.TangentChainSelect should be true")
	}
}
