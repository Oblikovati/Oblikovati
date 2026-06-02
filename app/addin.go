// SPDX-License-Identifier: GPL-2.0-only

package app

import "fmt"

// AddIn is a third-party (or first-party) extension — Inventor's
// ApplicationAddInServer. On Activate it contributes commands (which appear on the
// ribbon automatically), event handlers, and tools; Deactivate tears them down. The
// product is open from day one: the same flow registers built-in and external
// capabilities.
type AddIn interface {
	ID() string
	Activate(*Session) error
	Deactivate(*Session) error
}

// AddInManager loads and tracks add-ins — Inventor's ApplicationAddIns. Activation
// is idempotent per add-in.
type AddInManager struct {
	addins map[string]AddIn
	active map[string]bool
	order  []string
}

// NewAddInManager returns an empty add-in registry.
func NewAddInManager() *AddInManager {
	return &AddInManager{addins: map[string]AddIn{}, active: map[string]bool{}}
}

// Register makes an add-in known (not yet activated), erroring on a duplicate id.
func (m *AddInManager) Register(a AddIn) error {
	if _, dup := m.addins[a.ID()]; dup {
		return fmt.Errorf("app: add-in %q already registered", a.ID())
	}
	m.addins[a.ID()] = a
	m.order = append(m.order, a.ID())
	return nil
}

// Activate runs an add-in's Activate (no-op if already active).
func (m *AddInManager) Activate(s *Session, id string) error {
	a, ok := m.addins[id]
	if !ok {
		return fmt.Errorf("app: no add-in %q", id)
	}
	if m.active[id] {
		return nil
	}
	if err := a.Activate(s); err != nil {
		return err
	}
	m.active[id] = true
	return nil
}

// Deactivate runs an add-in's Deactivate (no-op if not active).
func (m *AddInManager) Deactivate(s *Session, id string) error {
	a, ok := m.addins[id]
	if !ok || !m.active[id] {
		return nil
	}
	if err := a.Deactivate(s); err != nil {
		return err
	}
	m.active[id] = false
	return nil
}

// IsActive reports whether an add-in is currently active.
func (m *AddInManager) IsActive(id string) bool { return m.active[id] }

// Registered returns the registered add-in ids in registration order.
func (m *AddInManager) Registered() []string {
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}
