// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// AddInLoadBehavior aliases the canonical Apache-2.0 enum (ADR-0018) so call sites
// here read naturally.
type AddInLoadBehavior = types.AddInLoadBehavior

const (
	LoadOnStartup = types.LoadOnStartup
	LoadOnDemand  = types.LoadOnDemand
	LoadDisabled  = types.LoadDisabled
)

// The registry slice of ApplicationAddIns is part of the public in-process contract.
var _ contract.AddInRegistry = (*AddInManager)(nil)

// AddInManifested is the optional capability of an [AddIn] that carries the C-ABI
// manifest JSON (a [wire.AddInManifest]) — every shared-library add-in does; a
// first-party in-process add-in may skip it and list with its id only.
type AddInManifested interface{ Manifest() string }

// AddInLocated is the optional capability of an [AddIn] that knows the
// shared-library file it was loaded from.
type AddInLocated interface{ Path() string }

// AddInAutomationProbe is the optional capability that reports whether an add-in's
// automation surface is actually present. The shared-library wrapper always has the
// CallAutomation method but may lack the ObkAddInAutomation export, so a plain
// interface assertion would over-report.
type AddInAutomationProbe interface{ HasAutomation() bool }

// AddInBehaviorStore persists the per-user load behaviors across sessions. A missing
// store file loads as an empty map (fresh install ⇒ everything LoadOnStartup).
type AddInBehaviorStore interface {
	Load() (map[string]types.AddInLoadBehavior, error)
	Save(map[string]types.AddInLoadBehavior) error
}

// UseBehaviorStore loads the persisted load behaviors and keeps the store so
// [AddInManager.SetLoadBehavior] writes through. Call before registering add-ins so
// startup activation respects the stored behaviors.
func (m *AddInManager) UseBehaviorStore(store AddInBehaviorStore) error {
	loaded, err := store.Load()
	if err != nil {
		return fmt.Errorf("app: load add-in behaviors: %w", err)
	}
	for id, b := range loaded {
		m.behaviors[id] = b
	}
	m.behaviorStore = store
	return nil
}

// LoadBehavior returns when the host activates the add-in on startup (the zero
// value, LoadOnStartup, for an id with no stored preference).
func (m *AddInManager) LoadBehavior(id string) types.AddInLoadBehavior {
	return m.behaviors[id]
}

// SetLoadBehavior records (and persists, when a store is wired) when the host should
// activate the add-in on future startups. It does not deactivate a running add-in.
func (m *AddInManager) SetLoadBehavior(id string, b types.AddInLoadBehavior) error {
	if _, ok := m.addins[id]; !ok {
		return fmt.Errorf("app: no add-in %q", id)
	}
	m.behaviors[id] = b
	if m.behaviorStore == nil {
		return nil
	}
	return m.behaviorStore.Save(m.behaviorSnapshot())
}

// behaviorSnapshot copies the non-default behaviors for persistence (storing the
// default would freeze it; omitting it lets a future default change apply).
func (m *AddInManager) behaviorSnapshot() map[string]types.AddInLoadBehavior {
	out := map[string]types.AddInLoadBehavior{}
	for id, b := range m.behaviors {
		if b != types.LoadOnStartup {
			out[id] = b
		}
	}
	return out
}

// Describe returns the registry entry for one add-in: its manifest identity (when it
// carries one) plus the host-side runtime state.
func (m *AddInManager) Describe(id string) (wire.AddInInfo, error) {
	a, ok := m.addins[id]
	if !ok {
		return wire.AddInInfo{}, fmt.Errorf("app: no add-in %q", id)
	}
	info := wire.AddInInfo{
		ID:            id,
		LoadBehavior:  m.behaviors[id],
		Activated:     m.active[id],
		HasAutomation: addInHasAutomation(a),
	}
	if located, ok := a.(AddInLocated); ok {
		info.Location = located.Path()
	}
	if manifested, ok := a.(AddInManifested); ok {
		fillFromManifest(&info, manifested.Manifest())
	}
	return info, nil
}

// fillFromManifest copies the manifest's identity fields into info. A malformed
// manifest leaves them empty rather than failing the listing — the id (from
// ObkAddInId) still identifies the entry.
func fillFromManifest(info *wire.AddInInfo, manifest string) {
	var man wire.AddInManifest
	if json.Unmarshal([]byte(manifest), &man) != nil {
		return
	}
	info.DisplayName, info.Version, info.Description = man.DisplayName, man.Version, man.Description
	info.Kind, info.Capabilities = man.Kind, man.Capabilities
}

// addInHasAutomation reports whether automation can actually be routed to a — the
// probe wins when present (shared-library wrappers), else the plain assertion.
func addInHasAutomation(a AddIn) bool {
	if probe, ok := a.(AddInAutomationProbe); ok {
		return probe.HasAutomation()
	}
	_, ok := a.(contract.AddInAutomation)
	return ok
}

// CallAutomation routes one automation request to the target add-in
// (ApplicationAddIn.Automation). The target must be active — automation is a live
// surface, not a manifest query — and must actually expose one.
func (m *AddInManager) CallAutomation(id, method string, args []byte) ([]byte, error) {
	a, ok := m.addins[id]
	if !ok {
		return nil, fmt.Errorf("app: no add-in %q", id)
	}
	if !m.active[id] {
		return nil, fmt.Errorf("app: add-in %q is not active; automation needs an active add-in", id)
	}
	auto, ok := a.(contract.AddInAutomation)
	if !ok || !addInHasAutomation(a) {
		return nil, fmt.Errorf("app: add-in %q exposes no automation surface", id)
	}
	return auto.CallAutomation(method, args)
}
