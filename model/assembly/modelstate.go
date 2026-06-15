// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "fmt"

// A model state (M12-F04) is a named tuple selecting one representation of each family — the
// single switch users flip. Activating it activates the three layers together. An empty family
// name leaves that family's active representation unchanged.

// modelState selects one representation of each family by name.
type modelState struct {
	id                      uint64
	name                    string
	active                  bool
	design, positional, lod string
}

// ID returns the model state's session id.
func (m *modelState) ID() uint64 { return m.id }

// Name returns the model state's display name.
func (m *modelState) Name() string { return m.name }

// Active reports whether this is the active model state.
func (m *modelState) Active() bool { return m.active }

// DesignViewName / PositionalName / LevelOfDetailName are the selected representation names.
func (m *modelState) DesignViewName() string    { return m.design }
func (m *modelState) PositionalName() string    { return m.positional }
func (m *modelState) LevelOfDetailName() string { return m.lod }

// CreateModelState adds a model state selecting one representation of each family by name.
func (r *Representations) CreateModelState(name, design, positional, lod string) ModelStateRep {
	m := &modelState{id: r.id(), name: repName(name, "ModelState", len(r.models)+1), design: design, positional: positional, lod: lod}
	r.models = append(r.models, m)
	return m
}

// ActivateModelState activates the three representations the model state selects (skipping a
// family whose name is empty or unresolved) and marks the model state active.
func (r *Representations) ActivateModelState(id uint64) (ModelStateRep, error) {
	m := r.modelByID(id)
	if m == nil {
		return nil, fmt.Errorf("assembly: no model state with id %d", id)
	}
	if d := r.designViewByName(m.design); d != nil {
		if _, err := r.ActivateDesignView(d.id); err != nil {
			return nil, err
		}
	}
	if p := r.positionalByName(m.positional); p != nil {
		if _, err := r.ActivatePositional(p.id); err != nil {
			return nil, err
		}
	}
	if l := r.lodByName(m.lod); l != nil {
		if _, err := r.ActivateLOD(l.id); err != nil {
			return nil, err
		}
	}
	for _, o := range r.models {
		o.active = o.id == id
	}
	return m, nil
}

// AllModelStates returns the model states in creation order (the host reads these to build the
// wire info rows); ModelStateByID looks one up (or nil).
func (r *Representations) AllModelStates() []ModelStateRep {
	out := make([]ModelStateRep, len(r.models))
	for i, m := range r.models {
		out[i] = m
	}
	return out
}

func (r *Representations) ModelStateByID(id uint64) ModelStateRep {
	if m := r.modelByID(id); m != nil {
		return m
	}
	return nil
}

func (r *Representations) modelByID(id uint64) *modelState {
	for _, m := range r.models {
		if m.id == id {
			return m
		}
	}
	return nil
}

// DeleteModelState removes a model state by id, reporting whether it was found.
func (r *Representations) DeleteModelState(id uint64) bool {
	for i, m := range r.models {
		if m.id == id {
			r.models = append(r.models[:i], r.models[i+1:]...)
			return true
		}
	}
	return false
}

// designViewByName / positionalByName / lodByName resolve a representation by name (empty name
// resolves to nil, so an unset model-state family is skipped).
func (r *Representations) designViewByName(name string) *designViewRep {
	for _, d := range r.design {
		if name != "" && d.name == name {
			return d
		}
	}
	return nil
}

func (r *Representations) positionalByName(name string) *positionalRep {
	for _, p := range r.pos {
		if name != "" && p.name == name {
			return p
		}
	}
	return nil
}

func (r *Representations) lodByName(name string) *lodRep {
	for _, l := range r.lod {
		if name != "" && l.name == name {
			return l
		}
	}
	return nil
}
